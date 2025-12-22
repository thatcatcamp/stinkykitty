.PHONY: build build-linux clean test install

# Build for current platform
build:
	CGO_ENABLED=1 go build -o stinky cmd/stinky/*.go

# Build for Linux (for VPS deployment)
build-linux:
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o stinky cmd/stinky/*.go

# Install to /usr/bin (requires sudo)
install: build
	sudo cp stinky /usr/bin/stinky
	sudo chmod +x /usr/bin/stinky

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -f stinky
	rm -f /tmp/stinky*

# Build and run locally
run: build
	./stinky server start

# Show help
help:
	@echo "Available targets:"
	@echo "  build        - Build for current platform with CGO enabled"
	@echo "  build-linux  - Cross-compile for Linux AMD64 (VPS deployment)"
	@echo "  install      - Build and install to /usr/bin"
	@echo "  test         - Run all tests"
	@echo "  clean        - Remove build artifacts"
	@echo "  run          - Build and run server locally"
	@echo "  help         - Show this help message"
