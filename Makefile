BINARY_NAME=slug
OS := $(shell uname)

run:
	go run ./cmd/app/ $(ARGS)


test:
	go test ./...


clean:
	rm -rf ./bin/$(BINARY_NAME)


build:
	mkdir -p ./bin
	go build -o ./bin/$(BINARY_NAME) ./cmd/app/
ifeq ($(OS), Darwin)
	codesign --sign - ./bin/$(BINARY_NAME)
endif


release:  clean
	mkdir -p ./bin
	go build -ldflags="-s -w" -o ./bin/$(BINARY_NAME) ./cmd/app/
ifeq ($(OS), Darwin)
	codesign --sign - ./bin/$(BINARY_NAME)
endif
