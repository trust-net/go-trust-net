DOCKER:=docker
#TAG:=p2p-pager
TAG:=poc-countr
docker:
ifeq ($(strip $(shell $(DOCKER) images $(TAG):latest -q)),)
	@echo "creating new '$(TAG):latest' from 'golang:latest' base"
	@echo "FROM golang" > Dockerfile
else
	@echo "re-using existing '$(TAG):latest' as base"
	@echo "FROM $(TAG)" > Dockerfile
endif
	@cat Dockerfile.base >> Dockerfile
	@docker build -t $(TAG) .
