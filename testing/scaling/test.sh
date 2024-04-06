#!/bin/bash

kubectl apply -f testing_1.yaml
sleep 300
kubectl apply -f testing_2.yaml
sleep 300
kubectl apply -f testing_3.yaml
sleep 300
kubectl apply -f testing_2.yaml
sleep 300
kubectl apply -f testing_1.yaml
sleep 300
kubectl delete -f testing_1.yaml