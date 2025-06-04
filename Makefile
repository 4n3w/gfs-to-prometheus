.PHONY: build clean test deps

BINARY=gfs-to-prometheus
GOARCH=amd64

build:
	go build -o ${BINARY} main.go

deps:
	go mod download
	go mod tidy

test:
	go test -v ./...

clean:
	go clean
	rm -f ${BINARY}

build-linux:
	GOOS=linux GOARCH=${GOARCH} go build -o ${BINARY}-linux-${GOARCH} main.go

build-darwin:
	GOOS=darwin GOARCH=${GOARCH} go build -o ${BINARY}-darwin-${GOARCH} main.go

build-windows:
	GOOS=windows GOARCH=${GOARCH} go build -o ${BINARY}-windows-${GOARCH}.exe main.go

all: build