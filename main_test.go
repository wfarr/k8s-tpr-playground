package main

import (
	"os"
	"testing"
	"time"

	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/unversioned"
	"k8s.io/client-go/1.5/pkg/fields"
	"k8s.io/client-go/1.5/pkg/runtime"
	"k8s.io/client-go/1.5/tools/cache"
)

func TestIntoWithSingle(t *testing.T) {
	client, err := buildClientFromFlags(os.Getenv("HOME") + "/.kube/config")
	if err != nil {
		panic(err.Error())
	}

	example := Example{}
	req := client.Get().
		Namespace("default").
		Resource("examples").
		Name("example1")
	err = req.Do().Into(&example)
	if err != nil {
		t.Fatal(err, req.URL().String())
	}

	if example.Spec.Foo == "" || example.Spec.Bar == false {
		t.Errorf("expected example to have non-default values, but got: %#v", example)
	}
}

func TestWithInformer(t *testing.T) {
	client, err := buildClientFromFlags(os.Getenv("HOME") + "/.kube/config")
	if err != nil {
		panic(err.Error())
	}

	schemeBuilder := runtime.NewSchemeBuilder(addKnownTypes)
	schemeBuilder.AddToScheme(api.Scheme)

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

	// go func() {
	for {
		select {
		case event := <-eventchan:
			if event.Spec.Foo == "" || event.Spec.Bar == false {
				t.Errorf("expected example to have non-default values, but got: %#v", event)
				close(stopchan)
				t.Fatal("stopping...")
			}

			t.Logf("%#v\n", event)
		}
	}
	// }()

}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(unversioned.GroupVersion{Group: "wfarr.systems", Version: "v1"},
		&Example{},
		&ExampleList{},
		&api.ListOptions{},
		&api.DeleteOptions{},
	)

	return nil
}
