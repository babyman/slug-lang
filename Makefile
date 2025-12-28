BINARY_NAME=slug
OS := $(shell uname)

BUILD_VER = Dev
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
COMMIT := $(shell git rev-parse --short HEAD)

run:
	# e.g. make run ARGS='--root ./tests --debug-ast ./tests/nil.slug'
	time go run ./cmd/app/main.go $(ARGS)

live:
	# requires `entr` see https://eradman.com/entrproject/
	find . \( -name "*.slug" -o -name "*.go" \) | entr -r time go run ./cmd/app/ $(ARGS)

stress:
	@for i in `seq 1 100`; do \
		echo "Run $$i"; \
		go run -race ./cmd/app/main.go -log-level none --root ./tests ./tests/boolean-logic.slug || exit 1; \
	done

test:
	# e.g. find . \( -name "*.slug" -o -name "*.go" \) | entr -r time make test
	go test ./... || exit 1
	@for file in $(shell find ./tests -name "*.slug" | sort); do \
		echo "Running tests for $$file"; \
		go run ./cmd/app/main.go -log-level error --root ./tests $$file || exit 1; \
	done
	go run ./cmd/app/main.go -log-level error --root . test \
		$(shell find './lib' -name "*.slug" | sed -e 's/\.\/lib\///' -e 's/\//./g' -e 's/\.slug//' | sort) \
		$(shell find './test-suites' -name "*.slug" | sort) || exit 1

lc: clean
	cloc  --exclude-dir=.idea --read-lang-def=slug_cloc_definition.txt .

clean:
	find ./ -name "*.ast.json" -type f -delete
	find ./ -name "*.ast.txt" -type f -delete
	rm -rf ./dist
	rm -rf ./bin/$(BINARY_NAME)


build:
	mkdir -p ./bin
	go build \
		-ldflags="-X main.Version=${BUILD_VER} -X main.BuildDate=${BUILD_DATE} -X main.Commit=${COMMIT}" \
		-o ./bin/$(BINARY_NAME) ./cmd/app/
ifeq ($(OS), Darwin)
	codesign --sign - ./bin/$(BINARY_NAME)
endif


release: clean
	mkdir -p ./bin
	go build \
		-ldflags="-s -w -X main.Version=${BUILD_VER} -X main.BuildDate=${BUILD_DATE} -X main.Commit=${COMMIT}" \
 		-o ./bin/$(BINARY_NAME) ./cmd/app/
ifeq ($(OS), Darwin)
	codesign --sign - ./bin/$(BINARY_NAME)
endif


windows: clean
	GOOS=windows GOARCH=amd64 go build \
		-ldflags="-s -w -X main.Version=${BUILD_VER} -X main.BuildDate=${BUILD_DATE} -X main.Commit=${COMMIT}" \
		-o ./bin/slug.exe ./cmd/app/
