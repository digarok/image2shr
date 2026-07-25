# image2shr — Apple IIgs Super Hi-Res image converter

VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -ldflags "-X github.com/digarok/image2shr/internal/cli.Version=$(VERSION)"
BIN      := bin/image2shr

.PHONY: build test lint cross clean

build:
	go build $(LDFLAGS) -o $(BIN) .

test:
	go test ./...

lint:
	@fmt_out=$$(gofmt -l .); \
	if [ -n "$$fmt_out" ]; then \
		echo "gofmt needed on:"; echo "$$fmt_out"; exit 1; \
	fi
	go vet ./...

# Cross-compile release binaries into dist/.
cross:
	@for os in darwin linux windows; do \
		for arch in amd64 arm64; do \
			ext=""; [ $$os = windows ] && ext=".exe"; \
			out="dist/image2shr-$$os-$$arch$$ext"; \
			echo "  $$out"; \
			GOOS=$$os GOARCH=$$arch go build $(LDFLAGS) -o $$out . || exit 1; \
		done; \
	done

clean:
	rm -rf bin dist
