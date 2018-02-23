#!/bin/bash

### make sure tag here matches Makefile declaration
TAG="poc-countr"

DOCKER="docker"

if [ $# -ne 2 ]; then
	echo "usage: $0 <port number> <boot node>"
	exit 1
fi

# check if TAG image exists to run
if [[ "$($DOCKER images $TAG:latest -q)" == "" ]]; then
	echo "ERROR: did not find docker image $TAG"
	exit 1
fi

# the bootnode address should either be publicly reachable peer, or a peer running
# on same local subnet as the container (e.g. same bridge network)
# e.g. enode://<pub key>@172.17.0.2:30303, or
# e.g. enode://<pub key>@192.168.1.114:30303

docker run -it --rm --publish $1:30303 \
	--name node-$1 \
	--entrypoint /go/bin/app \
	$TAG -name Node@$1 -bootnode $2
