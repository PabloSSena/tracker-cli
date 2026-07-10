BINARY  := tracker
DIST    := dist

# Build puro-Go (sem CGO) — o driver modernc.org/sqlite dispensa gcc.
export CGO_ENABLED := 0

.PHONY: all build linux linux-arm64 install run test vet clean

## build para o SO/arquitetura atual
all: build
build:
	go build -o $(BINARY) ./

## binário Linux amd64
linux:
	GOOS=linux GOARCH=amd64 go build -o $(DIST)/$(BINARY)-linux-amd64 ./

## binário Linux arm64 (ex: Raspberry Pi, servidores ARM)
linux-arm64:
	GOOS=linux GOARCH=arm64 go build -o $(DIST)/$(BINARY)-linux-arm64 ./

## instala em ~/.local/bin (precisa estar no PATH); rode com: tracker
install: linux
	install -Dm755 $(DIST)/$(BINARY)-linux-amd64 $(HOME)/.local/bin/$(BINARY)
	@echo "Instalado em $(HOME)/.local/bin/$(BINARY) — garanta que ~/.local/bin está no PATH."

## roda a CLI direto do código
run:
	go run ./

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -rf $(DIST) $(BINARY)
