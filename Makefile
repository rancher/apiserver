test:
	go test -race -cover ./...

lint:
	golangci-lint run

.PHONY: test
