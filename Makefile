build:
	go build -o ./pctl cmd/pctl/main.go

test: unit-test integration-test

unit-test:
	go test ./pkg/...

integration-test:
	go test ./tests/...

lint:
	golangci-lint run