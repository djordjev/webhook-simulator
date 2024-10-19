BINARY_NAME=wh-simulator

.PHONY: build clean

build:
	go build -o ./_out/${BINARY_NAME} ./cmd/wh-simulator.go

clean:
	rm -rf ./_out