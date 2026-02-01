.PHONY: test test-unit test-integration build clean

# Run all tests
test: test-unit test-integration clean

# Run unit tests only
test-unit:
	go test -v -race $$(go list ./... | grep -v /tests)

# Run integration tests
test-integration:
	DB_PATH=:memory: \
	MIGRATIONS_PATH=file://$(CURDIR)/db/migrations \
	JOBS_INTERVAL=1s \
	go test -v -race ./tests/...

# Build the application
build:
	go build -o bin/gniot ./cmd/gniot

# Clean build artifacts
clean:
	rm -f tests/gniot.db
