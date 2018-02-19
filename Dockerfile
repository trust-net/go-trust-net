# use this first time to get dependencies and tag it as p2p-pager image
#FROM golang

# use this for incremental build on top of last p2p-pager image
FROM p2p-pager

# uncomment below to cleanup older image (e.g. when directory layout changes)
RUN rm -rf /go/src/github.com/trust-net/go-trust-net

# copy current source codebase
ADD . /go/src/github.com/trust-net/go-trust-net

# uncomment these during first build, to install dependencies on golang image
#RUN go get github.com/ethereum/go-ethereum
#RUN go get github.com/satori/go.uuid

# build and install app
RUN go clean -i -x
RUN go install github.com/trust-net/go-trust-net/app

ENTRYPOINT /go/bin/app

EXPOSE 30303
