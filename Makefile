GOPATH := `pwd`/vendor:$(GOPATH)

build:
	GOPATH=$(GOPATH) go build main.go

run:
	GOPATH=$(GOPATH) go run main.go

test:
	GOPATH=$(GOPATH) go test
