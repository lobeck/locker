GO ?= go
GOPATH := $(CURDIR)/../../..
PACKAGES := $(shell GOPATH=$(GOPATH) go list ./... | grep -v /vendor/)

build:
	rm locker
	GOPATH=$(GOPATH) GOOS=linux GOARCH=amd64 $(GO) build

fmt:
	GOPATH=$(GOPATH) find . -name "*.go" | xargs gofmt -w

vet:
	GOPATH=$(GOPATH) $(GO) vet $(PACKAGES)