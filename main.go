package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/unversioned"
	"k8s.io/client-go/1.5/pkg/runtime"
	"k8s.io/client-go/1.5/pkg/runtime/serializer"
	"k8s.io/client-go/1.5/rest"
	"k8s.io/client-go/1.5/tools/clientcmd"

	_ "k8s.io/client-go/1.5/plugin/pkg/client/auth/gcp"
)

type ExampleSpec struct {
	Foo string `json:"foo"`
	Bar bool   `json:"bar"`
}

type Example struct {
	runtime.TypeMeta `json:",inline"`
	api.ObjectMeta   `json:"metadata"`

	Spec ExampleSpec `json:"spec"`
}

type ExampleList struct {
	runtime.TypeMeta     `json:",inline"`
	unversioned.ListMeta `json:"metadata"`

	Items []Example `json:"items"`
}

func (e *Example) UnmarshalText(data []byte) error {
	return json.Unmarshal(data, e)
}

func (e *Example) MarshalText() ([]byte, error) {
	return json.Marshal(e)
}

func (e *Example) GetObjectKind() unversioned.ObjectKind {
	return &e.TypeMeta
}

func (el *ExampleList) UnmarshalText(data []byte) error {
	return json.Unmarshal(data, el)
}

func (el *ExampleList) MarshalText() ([]byte, error) {
	return json.Marshal(el)
}

func (el *ExampleList) GetObjectKind() unversioned.ObjectKind {
	return &el.TypeMeta
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
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: api.Codecs}

	return rest.RESTClientFor(config)
}

func configFromFlags(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}
