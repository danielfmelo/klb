#!/bin/bash

set -o errexit
set -o nounset

DOCKER_ENV=$(mktemp run-docker-env.XXXXXXX)
trap "rm -f $DOCKER_ENV" EXIT

GOPATH=/go
WORKDIR=$GOPATH/src/github.com/NeowayLabs/klb

echo GOPATH=$GOPATH >> $DOCKER_ENV
echo AZURE_SUBSCRIPTION_ID=$AZURE_SUBSCRIPTION_ID >> $DOCKER_ENV
echo AZURE_TENANT_ID=$AZURE_TENANT_ID >> $DOCKER_ENV
echo AZURE_CLIENT_ID=$AZURE_CLIENT_ID >> $DOCKER_ENV
echo AZURE_CLIENT_SECRET=$AZURE_CLIENT_SECRET >> $DOCKER_ENV
echo AZURE_SUBSCRIPTION_NAME=$AZURE_SUBSCRIPTION_NAME >> $DOCKER_ENV
echo AZURE_SERVICE_PRINCIPAL=$AZURE_SERVICE_PRINCIPAL >> $DOCKER_ENV
