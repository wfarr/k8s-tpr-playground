package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/meta"
	"k8s.io/client-go/1.5/pkg/api/unversioned"
	"k8s.io/client-go/1.5/pkg/fields"
	"k8s.io/client-go/1.5/pkg/runtime"
	"k8s.io/client-go/1.5/pkg/runtime/serializer"
	"k8s.io/client-go/1.5/rest"
	"k8s.io/client-go/1.5/tools/cache"
	"k8s.io/client-go/1.5/tools/clientcmd"

	_ "k8s.io/client-go/1.5/plugin/pkg/client/auth/gcp"
)

type ExampleSpec struct {
	Foo string `json:"foo"`
	Bar bool   `json:"bar"`
}

type Example struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata             api.ObjectMeta `json:"metadata"`

	Spec ExampleSpec `json:"spec"`
}

type ExampleList struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata             unversioned.ListMeta `json:"metadata"`

	Items []Example `json:"items"`
}

func (e *Example) GetObjectKind() unversioned.ObjectKind {
	return &e.TypeMeta
}

func (e *Example) GetObjectMeta() meta.Object {
	return &e.Metadata
}

func (el *ExampleList) GetObjectKind() unversioned.ObjectKind {
	return &el.TypeMeta
}

func (el *ExampleList) GetListMeta() unversioned.List {
	return &el.Metadata
}

type ExampleListCopy ExampleList
type ExampleCopy Example

func (e *Example) UnmarshalJSON(data []byte) error {
	tmp := ExampleCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := Example(tmp)
	*e = tmp2
	return nil
}

func (el *ExampleList) UnmarshalJSON(data []byte) error {
	tmp := ExampleListCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := ExampleList(tmp)
	*el = tmp2
	return nil
}

var (
	config *rest.Config
)

func main() {
	kubeconfig := flag.String("kubeconfig", "", "the path to a kubeconfig, specifies this tool runs outside the cluster")
	flag.Parse()

	client, err := buildClientFromFlags(*kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	exampleList := ExampleList{}
	err = client.Get().
		Resource("examples").
		Do().Into(&exampleList)
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("%#v\n", exampleList)

	example := Example{}
	err = client.Get().
		Namespace("default").
		Resource("examples").
		Name("example1").
		Do().Into(&example)
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("%#v\n", example)

	fmt.Println()
	fmt.Println("---------------------------------------------------------")
	fmt.Println()

	eventchan := make(chan *Example)
	stopchan := make(chan struct{}, 1)
	source := cache.NewListWatchFromClient(client, "examples", api.NamespaceAll, fields.Everything())

	createDeleteHandler := func(obj interface{}) {
		example := obj.(*Example)
		eventchan <- example
	}

	updateHandler := func(old interface{}, obj interface{}) {
		example := obj.(*Example)
		eventchan <- example
	}

	_, controller := cache.NewInformer(
		source,
		&Example{},
		time.Second*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc:    createDeleteHandler,
			UpdateFunc: updateHandler,
			DeleteFunc: createDeleteHandler,
		})

	go controller.Run(stopchan)

	go func() {
		for {
			select {
			case event := <-eventchan:
				fmt.Printf("%#v\n", event)
			}
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	for {
		select {
		case s := <-signals:
			fmt.Printf("received signal %#v, exiting...\n", s)
			os.Exit(0)
		}
	}
}

func buildClientFromFlags(kubeconfig string) (*rest.RESTClient, error) {
	config, err := configFromFlags(kubeconfig)
	if err != nil {
		return nil, err
	}

	config.GroupVersion = &unversioned.GroupVersion{
		Group:   "wfarr.systems",
		Version: "v1",
	}
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: api.Codecs}

	schemeBuilder := runtime.NewSchemeBuilder(addKnownTypes)
	schemeBuilder.AddToScheme(api.Scheme)

	return rest.RESTClientFor(config)
}

func configFromFlags(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(
		unversioned.GroupVersion{Group: "wfarr.systems", Version: "v1"},
		&Example{},
		&ExampleList{},
		&api.ListOptions{},
		&api.DeleteOptions{},
	)

	return nil
}
