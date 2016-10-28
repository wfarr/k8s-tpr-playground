package main

import (
	"os"
	"testing"
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
