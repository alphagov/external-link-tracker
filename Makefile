.PHONY: build run test clean

GOPATH := `pwd`/vendor:$(GOPATH)
BINARY := external-link-tracker
BUILDFILES := main.go handlers.go

build:
	GOPATH=$(GOPATH) go build -o $(BINARY) $(BUILDFILES)

run:
	GOPATH=$(GOPATH) go run $(BUILDFILES)

test:
	LINK_TRACKER_MONGO_DB=external_link_tracker_test GOPATH=$(GOPATH) go test
