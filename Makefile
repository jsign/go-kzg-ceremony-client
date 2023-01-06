lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.50.1 run
.PHONY: lint

test:
	go test ./... -race
.PHONY: test

build:
	go build -o kzgcli ./cmd/kzgcli
.PHONY: build