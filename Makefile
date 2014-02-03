.PHONY: build run test clean

GOPATH := `pwd`/vendor:$(GOPATH)
BINARY := external-link-tracker

build:
	GOPATH=$(GOPATH) go build -o $(BINARY) main.go

run:
	GOPATH=$(GOPATH) go run main.go

test:
	LINK_TRACKER_MONGO_DB=external_link_tracker_test GOPATH=$(GOPATH) go test
