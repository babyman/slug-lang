BINARY_NAME=slug
OS := $(shell uname)

run:
	# e.g. make run ARGS='--root ./tests --debug-ast ./tests/nil.slug'
	time go run ./cmd/app/ $(ARGS)

live:
	# requires `entr` see https://eradman.com/entrproject/
	find . \( -name "*.slug" -o -name "*.go" \) | entr -r time go run ./cmd/app/ $(ARGS)

test: release
	go test ./... || exit 1
	@for file in $(shell find ./tests -name "*.slug" | sort); do \
		echo "Running tests for $$file"; \
		./bin/$(BINARY_NAME) --root ./tests $$file || exit 1; \
	done
	slug test slug.math slug.std slug.list

lc: clean
	cloc  --exclude-dir=.idea --read-lang-def=slug_cloc_definition.txt .

clean:
	find ./lib -name "*.ast.json" -type f -delete
	find ./tests -name "*.ast.json" -type f -delete
	rm -rf ./dist
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


package: release windows
	mkdir -p ./dist/slug/bin ./dist/slug/lib ./dist/slug/docs
	cp -r ./bin/* ./dist/slug/bin/
	cp -r ./lib/* ./dist/slug/lib/ 2>/dev/null || :
	cp -r ./docs/* ./dist/slug/docs/ 2>/dev/null || :
	cp readme.md ./dist/slug/ 2>/dev/null || :
ifeq ($(OS), Darwin)
	cd ./dist && zip -r slug.zip slug
else
	cd ./dist && zip -r slug.zip slug
endif
