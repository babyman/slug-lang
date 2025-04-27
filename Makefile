BINARY_NAME=slug
OS := $(shell uname)

run:
	# e.g. make run ARGS='--root ./tests --debug-ast ./tests/nil.slug'
	go run ./cmd/app/ $(ARGS)


test: release
	go test ./... || exit 1
	@for file in $(shell find ./tests -name "*.slug"); do \
		echo "Running tests for $$file"; \
		./bin/$(BINARY_NAME) --root ./tests $$file || exit 1; \
	done


clean:
	rm -rf ./bin/$(BINARY_NAME)


build:
	mkdir -p ./bin
	go build -o ./bin/$(BINARY_NAME) ./cmd/app/
ifeq ($(OS), Darwin)
	codesign --sign - ./bin/$(BINARY_NAME)
endif


release: clean
	mkdir -p ./bin
	go build -ldflags="-s -w" -o ./bin/$(BINARY_NAME) ./cmd/app/
ifeq ($(OS), Darwin)
	codesign --sign - ./bin/$(BINARY_NAME)
endif


windows: clean
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o ./bin/slug.exe ./cmd/app/
