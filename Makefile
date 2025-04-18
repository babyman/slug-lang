BINARY_NAME=slug

run:
	go run ./

test:
	go test ./...

clean:
	rm -rf ./bin/$(BINARY_NAME)

build:
	go build -o ./bin/$(BINARY_NAME) ./
	codesign --sign - ./bin/$(BINARY_NAME)

release:
	go build -ldflags="-s -w" -o ./bin/$(BINARY_NAME) ./
	codesign --sign - ./bin/$(BINARY_NAME)
