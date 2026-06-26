.PHONY: all build build-web build-server run dev clean test deps help init

# Default target
all: build

# Build everything
build: build-web build-server

# Build web frontend
build-web:
	@echo "Building web frontend..."
	cd web && npm install && npm run build

# Build Go server
build-server:
	@echo "Building Go server..."
	go mod download
	go build -o bin/anox-server ./cmd/anox-server

# Run the server (requires built web assets)
run: build
	@echo "Starting Anox server..."
	./bin/anox-server

# Development mode - run server only
dev-server:
	@echo "Starting Anox server in development mode..."
	go run ./cmd/anox-server

# Development mode - run web only
dev-web:
	@echo "Starting web development server..."
	cd web && npm install && npm run dev

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -rf web/dist/
	rm -rf web/node_modules/

# Run tests
test:
	@echo "Running tests..."
	go test ./...

# Download dependencies
deps:
	@echo "Downloading Go dependencies..."
	go mod download
	@echo "Installing npm dependencies..."
	cd web && npm install

# Create necessary directories
init:
	@echo "Creating data directories..."
	mkdir -p data/configs
	mkdir -p logs

# Help
help:
	@echo "Anox - Microservice Governance Platform"
	@echo ""
	@echo "Available targets:"
	@echo "  make build       - Build both web and server"
	@echo "  make build-web   - Build web frontend only"
	@echo "  make build-server- Build Go server only"
	@echo "  make run         - Build and run the server"
	@echo "  make dev-server  - Run Go server in development mode"
	@echo "  make dev-web     - Run web development server"
	@echo "  make clean       - Clean build artifacts"
	@echo "  make test        - Run tests"
	@echo "  make deps        - Download dependencies"
	@echo "  make init        - Create necessary directories"
	@echo "  make help        - Show this help"
