.PHONY: clean all fmt test binary

all: binary

docker-wicket: $(shell find . -type f -name '*.go')
	CGO_ENABLED=0 go build -a -installsuffix cgo

binary: docker-wicket

vet:
	go vet ./...

test:
	go test ./...

fmt:
	@test -z $(shell gofmt -s -l .) || echo "gofmt -s"
