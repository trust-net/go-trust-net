#!/bin/bash

### make sure tag here matches Makefile declaration
TAG="poc-countr"

DOCKER="docker"

if [ $# -ne 2 ]; then
	echo "usage: $0 <port number> <config file>"
	exit 1
fi

# check if TAG image exists to run
if [[ "$($DOCKER images $TAG:latest -q)" == "" ]]; then
	echo "ERROR: did not find docker image $TAG"
	exit 1
fi

### mount points for data directories
DATA_DIR="/tmp/data"
mkdir -p $DATA_DIR
### copy the config file for this node into mount point
cp $2 $DATA_DIR/node-$1.json

docker run -it --rm --publish $1:30303 \
	--name node-$1 \
	--mount type=bind,source="$DATA_DIR",target=/tmp \
	--entrypoint /go/bin/app \
	$TAG -file /tmp/node-$1.json
