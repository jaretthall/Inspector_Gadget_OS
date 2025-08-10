# Inspector Gadget OS - Phase 0 Build System

.PHONY: all clean gadget-cli web-build o-llama-build test help

# Default target
all: gadget-cli web-build o-llama-build

# Build gadget-framework CLI
gadget-cli:
	@echo "Building gadget-framework..."
	cd gadget-framework && go build -o ../bin/go-go-gadget ./cmd/go-go-gadget

# Build web management server
web-build:
	@echo "Building web management server..."
	cd web && go build -o ../bin/web-server .

# Build o-llama enhanced server
o-llama-build:
	@echo "Building o-llama enhanced server..."
	cd o-llama && go build -o ../bin/ollama-server ./cmd/ollama-server

# Run tests for all modules
test:
	@echo "Running tests for all modules..."
	cd gadget-framework && go test ./...
	cd web && go test ./...
	cd o-llama && go test ./...

# Create bin directory
bin:
	mkdir -p bin

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	cd gadget-framework && go clean
	cd web && go clean
	cd o-llama && go clean

# Show help
help:
	@echo "Inspector Gadget OS - Phase 0 Build Commands"
	@echo ""
	@echo "Available targets:"
	@echo "  all           - Build all modules (default)"
	@echo "  gadget-cli    - Build gadget-framework CLI"
	@echo "  web-build     - Build web management server" 
	@echo "  o-llama-build - Build o-llama enhanced server"
	@echo "  test          - Run tests for all modules"
	@echo "  clean         - Remove build artifacts"
	@echo "  help          - Show this help message"

# Ensure bin directory exists for all build targets
gadget-cli web-build o-llama-build: | bin