.PHONY: fmt test tidy

fmt:
	gofmt -w $$(find sdk -name '*.go')

tidy:
	go mod tidy

test:
	go test ./...
