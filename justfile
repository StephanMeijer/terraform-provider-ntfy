# Default recipe - build the provider
default: build

# Build the provider binary
build:
    go build -o terraform-provider-ntfy

# Build and install to ~/go/bin/
install: build
    mv terraform-provider-ntfy ~/go/bin/

# Run go vet for static analysis
vet:
    go vet ./...

# Run all tests
test:
    go test -v ./...

# Run tests with race detection
test-race:
    go test -v -race ./...

# Check code formatting
fmt-check:
    @if [ -n "$(gofmt -l .)" ]; then \
        echo "The following files need formatting:"; \
        gofmt -l .; \
        exit 1; \
    fi
    @echo "All files formatted correctly"

# Format all Go files
fmt:
    gofmt -w .

# Run all quality checks (fmt, vet, test)
check: fmt-check vet test

# Clean build artifacts
clean:
    rm -f terraform-provider-ntfy
    rm -rf dist/

# Build for all platforms (requires GoReleaser)
release-snapshot:
    goreleaser release --snapshot --clean

# Show help
help:
    @just --list

# Stop and clean up test infrastructure
test-down:
    docker compose down -v

# Run acceptance tests (requires Docker ntfy running)
test-acc:
    @docker compose up -d
    @bash scripts/setup-test-ntfy.sh
    TF_ACC=1 NTFY_URL=http://localhost:8080 NTFY_USERNAME=admin NTFY_PASSWORD=admin go test -v -timeout 300s -run TestAcc ./...
    @docker compose down -v

# Run all tests (unit + acceptance)
test-all: test test-acc

# Generate documentation
docs:
    go generate ./...

# Check documentation is up to date
docs-check:
    go generate ./...
    @if [ -n "$(git status --porcelain docs/)" ]; then \
        echo "Documentation is out of date. Run 'just docs' and commit changes."; \
        git status docs/; \
        exit 1; \
    fi
    @echo "Documentation is up to date"
