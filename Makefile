BINARY := autoenv
CGO := CGO_ENABLED=1

.PHONY: build clean test install

build:
	$(CGO) go build -o $(BINARY) .

clean:
	rm -f $(BINARY)

test:
	$(CGO) go test ./...

install: build
	cp $(BINARY) $(GOPATH)/bin/$(BINARY)
