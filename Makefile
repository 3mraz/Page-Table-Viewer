# Makefile for Go project with Tailwind CSS

# Variables
CSS_INPUT=static/css/styles.css
CSS_OUTPUT=static/css/output.css
GO_FILES=./cmd/web/*.go

# Default target
all: build

# Build Tailwind CSS
build-css:
	@echo "Building Tailwind CSS..."
	npm run build
	@echo "Build completed!"

# run-css:
# 	@echo "Running Tailwind CSS..."
# 	npm run watch 

build-src:
	@echo "Building PTEditor..."
	cd src && make
	@echo "Build completed!"

# Build Go project
build: build-css build-src
	@echo "Building Go project..."
	go build main.go 
	@echo "Build completed!"


# Clean up build artifacts
clean:
	@echo "Cleaning up..."
	rm -f main 
	cd src && make clean
