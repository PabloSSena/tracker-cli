BINARY  := tracker
DIST    := dist

# Pure-Go build (no CGO) — the modernc.org/sqlite driver doesn't need gcc.
export CGO_ENABLED := 0

# ".exe" on Windows, empty on other OSes.
GOEXE := $(shell go env GOEXE)

.PHONY: all build linux linux-arm64 windows windows-arm64 install run test vet clean

## build for the current OS/architecture
all: build
build:
	go build -o $(BINARY)$(GOEXE) ./

## Linux amd64 binary
linux:
	GOOS=linux GOARCH=amd64 go build -o $(DIST)/$(BINARY)-linux-amd64 ./

## Linux arm64 binary (e.g. Raspberry Pi, ARM servers)
linux-arm64:
	GOOS=linux GOARCH=arm64 go build -o $(DIST)/$(BINARY)-linux-arm64 ./

## Windows amd64 binary
windows:
	GOOS=windows GOARCH=amd64 go build -o $(DIST)/$(BINARY)-windows-amd64.exe ./

## Windows arm64 binary
windows-arm64:
	GOOS=windows GOARCH=arm64 go build -o $(DIST)/$(BINARY)-windows-arm64.exe ./

## install to $(shell go env GOPATH)/bin — works on Windows, Linux and macOS
install:
	go install ./
	@echo "Installed to $(shell go env GOPATH)/bin/$(BINARY)$(GOEXE) — make sure that folder is on your PATH."

## run the CLI straight from source
run:
	go run ./

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -rf $(DIST) $(BINARY) $(BINARY).exe
