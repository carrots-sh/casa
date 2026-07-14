MODERNIZE := golang.org/x/tools/go/analysis/passes/modernize/cmd/modernize@latest

.PHONY: build fmt modernize lint test e2e check

build:
	go build -o casa ./cmd/casa

fmt:
	go fmt ./...

# rewrite code to modern Go idioms (https://go.dev/blog/gofix)
modernize:
	go run $(MODERNIZE) -fix -test ./...

lint:
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@latest ./...
	go run $(MODERNIZE) -test ./...

test:
	go test ./...

# every casa action, driven end-to-end in a sandbox (real chezmoi/git/age)
e2e:
	./scripts/e2e.sh

check: fmt lint test e2e
