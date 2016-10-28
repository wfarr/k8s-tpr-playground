#!/bin/bash

set -eux

kubectl apply -f extensions/
kubectl apply -f examples/

kubectl get tests

go test -v .