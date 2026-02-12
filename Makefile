BINARY := autoenv
CGO := CGO_ENABLED=1

.PHONY: build clean test lint install ci release

build:
	$(CGO) go build -trimpath -o $(BINARY) .

clean:
	rm -f $(BINARY)

test:
	$(CGO) go test -race -count=1 ./...

lint:
	golangci-lint run ./...

install: build
	cp $(BINARY) $(GOPATH)/bin/$(BINARY)

ci: lint test build

release:
	goreleaser release --clean

dagger:
	cd ci && go run main.go
