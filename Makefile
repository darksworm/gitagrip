.PHONY: test test-e2e build clean

# Run regular unit tests
test:
	go test ./...

# Run E2E tests with build tags
test-e2e:
	go test -tags e2e ./e2e -v

# Build the application
build:
	go build -o gitagrip .

# Clean build artifacts
clean:
	go clean
	rm -f gitagrip
	rm -f e2e/gitagrip_e2e