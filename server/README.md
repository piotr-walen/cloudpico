Install go 1.25.6
```
https://go.dev/doc/install
```

Build server with linker flags
```
APP_ENV=prod go build -o bin/cloudpico-server -ldflags "-X main.version=1.2.3" ./... 
```

Install linter locally
```
https://golangci-lint.run/docs/welcome/install/local/
```

Run linter
```
golangci-lint run ./...
```

Run tests
```
go test ./...
```

Run tests with coverage
```
go test -coverpkg=./... -coverprofile=coverage.out -covermode=atomic -timeout 2m ./...
```

Display coverage
```
go tool cover -func=coverage.out
```