#!/bin/bash
if [ $# -ne 2 ]; then
	echo "usage: $0 <port number> <boot node>"
	exit 1
fi

# the bootnode address should either be publicly reachable peer, or a peer running
# on same local subnet as the container (e.g. same bridge network)
# e.g. enode://<pub key>@172.17.0.2:30303, or
# e.g. enode://<pub key>@192.168.1.114:30303

docker run -it --rm --publish $1:30303 \
	--name node-$1 \
	--entrypoint /go/bin/pager \
	p2p-pager -name Node@$1 -bootnode $2
