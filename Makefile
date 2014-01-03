GOPATH := `pwd`/vendor:$(GOPATH)

build:
	GOPATH=$(GOPATH) go build -o external-link-tracker main.go

run:
	GOPATH=$(GOPATH) go run main.go

test:
	GOPATH=$(GOPATH) go test
