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
	unversioned.TypeMeta `json:",inline"`
	// Problem 1:
	// If ObjectMeta is embedded, because CodecEncodeSelf() is defined for
	// ObjectMeta in
	// https://github.com/kubernetes/client-go/blob/v1.5.0/1.5/pkg/api/types.generated.go,
	// Example will inherit the CodecEncodeSelf(), and ugorji won't call
	// Example.UnmarshalJSON won't be called.
	Metadata api.ObjectMeta `json:"metadata"`

	Spec ExampleSpec `json:"spec"`
}

type ExampleList struct {
	//TypeMeta unversioned.TypeMeta `json:",inline"`
	//Metadata unversioned.ListMeta `json:"metadata"`
	runtime.TypeMeta     `json:",inline"`
	unversioned.ListMeta `json:"metadata"`

	Items []Example `json:"items"`
}

// Problem 2:
// Looks like ugorji has bug when using UnmarhsalText. Always get error:
// json: expect char '"' but got char '{'. So I have to workaround and use
// UnmarshalJSON.

// func (e *Example) UnmarshalText(data []byte) error {
// 	fmt.Println("CHAO: calling Unmarshal Example")
// 	return json.Unmarshal(data, e)
// }
//
// func (e *Example) MarshalText() ([]byte, error) {
// 	return json.Marshal(e)
// }

func (e *Example) GetObjectKind() unversioned.ObjectKind {
	return &e.TypeMeta
}

type ExampleList2 ExampleList

func (el *ExampleList) UnmarshalJSON(data []byte) error {
	fmt.Println("CHAO: calling UnmarshalJSON ExampleList")
	fmt.Printf("CHAO: data=%s\n", data)
	tmp := ExampleList2{}
	// Workaround: has to call json.Unmarshal on ExampleList2, otherwise it will
	// be an endless loop.
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := ExampleList(tmp)
	*el = tmp2
	return nil
}

type Example2 Example

func (el *Example) UnmarshalJSON(data []byte) error {
	fmt.Println("CHAO: calling UnmarshalJSON Example")
	fmt.Printf("CHAO: data=%s\n", data)
	tmp := Example2{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := Example(tmp)
	*el = tmp2
	return nil
}

// func (el *ExampleList) UnmarshalText(data []byte) error {
// 	fmt.Println("CHAO: calling UnmarshalText ExampleList")
// 	fmt.Printf("CHAO: data=%s\n", data)
// 	return json.Unmarshal(data, el)
// }
//
// func (el *ExampleList) MarshalText() ([]byte, error) {
// 	return json.Marshal(el)
// }

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
	config.ContentType = runtime.ContentTypeJSON

	return rest.RESTClientFor(config)
}

func configFromFlags(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}
