unit-test:
    go test -v ./...

lint:
    "$(go env GOPATH)"/bin/golangci-lint run
