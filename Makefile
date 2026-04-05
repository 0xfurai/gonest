.PHONY: build test test-verbose test-cover test-race lint fmt vet clean

build:
	go build ./...

test:
	go test ./...

test-verbose:
	go test -v ./...

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

test-race:
	go test -race ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

lint: fmt vet

clean:
	rm -f coverage.out
