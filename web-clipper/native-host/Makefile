VERSION ?= 0.1.0
BIN     := bin/org-clipper-host

.PHONY: build test vet cover clean

build:
	go build -trimpath -ldflags "-X main.version=$(VERSION)" -o $(BIN) ./cmd/org-clipper-host

test:
	go test -count=1 ./...

vet:
	go vet ./...

cover:
	go test -count=1 -cover ./...

clean:
	rm -rf bin
