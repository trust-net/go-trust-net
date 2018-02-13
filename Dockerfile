# use this first time to get dependencies and tag it as p2p-pager image
#FROM golang

# use this for incremental build on top of last p2p-pager image
FROM p2p-pager

ADD . /go/src/github.com/trust-net/go-trust-net

# uncomment these for first build, to create p2p-pager image
#RUN go get github.com/ethereum/go-ethereum
#RUN go get github.com/satori/go.uuid

RUN go install github.com/trust-net/go-trust-net/app

ENTRYPOINT /go/bin/app

EXPOSE 30303
