.PHONY: build test smoke clean

build:
	go build -o ./bin/redline ./cmd/redline

test:
	go test ./...

smoke:
	out=$$(mktemp -d); \
	go run ./cmd/redline reveal workspace/sample.docx --output $$out; \
	test -f $$out/manifest.json; \
	test -f $$out/comments.md; \
	test -f $$out/review-intent.json; \
	test -f $$out/source-model.json; \
	test -d $$out/sections; \
	echo "smoke workspace: $$out"

clean:
	rm -f ./bin/redline
