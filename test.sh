#!/bin/bash

set -eux

kubectl apply -f extensions/
kubectl apply -f examples/

kubectl get examples example1

go test -v .