.PHONY: build run test clean

GOPATH := `pwd`/vendor:$(GOPATH)
BINARY := external-link-tracker

build:
	GOPATH=$(GOPATH) go build -o $(BINARY) main.go

run:
	GOPATH=$(GOPATH) go run main.go

test:
	GOPATH=$(GOPATH) go test
