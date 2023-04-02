#!/bin/bash

kubectl create secret docker-registry docker-registry-credentials \
  --namespace=default \
  --docker-server=https://index.docker.io/v1/ \
  --docker-username=$USERNAME \
  --docker-password=$PASSWORD \
  --docker-email=$EMAIL

