lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.50.1 run
.PHONY: lint

test:
	go test ./... -race
.PHONY: test

build:
	go build ./cmd/kzgcli
.PHONY: build

small-linux-arm-build:
	GOOS=linux GOARCH=arm go build -ldflags="-s -w" ./cmd/kzgcli
	upx --brute kzgcli
.PHONY: small-linux-build

bench:
	go test ./... -run=none -bench=.
.PHONY: bench