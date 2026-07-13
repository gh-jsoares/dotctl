VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -ldflags "-X github.com/gh-jsoares/dotctl/cmd.version=$(VERSION) -X github.com/gh-jsoares/dotctl/cmd.commit=$(COMMIT)"

.PHONY: build install test clean release man

build:
	go build $(LDFLAGS) -o dotctl .

install: build
	cp dotctl ~/.local/bin/dotctl

test:
	go test ./...

clean:
	rm -f dotctl
	rm -rf dist/

man: build
	./dotctl man --dir man

release: clean
	mkdir -p dist
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/dotctl_darwin_arm64 .
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/dotctl_darwin_amd64 .
