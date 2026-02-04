.PHONY: build install clean install-whisper download-model test lint fmt setup-hooks

# Paths
WHISPER_DIR := $(HOME)/.local/share/whisper.cpp
MODEL_DIR := $(HOME)/.cache/whisper
MODEL_PATH := $(MODEL_DIR)/ggml-base.bin
BIN_DIR := bin

# Build the memorex binary
build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/memorex ./cmd/memorex

# Install to GOPATH/bin
install: build
	go install ./cmd/memorex

# Clean build artifacts
clean:
	rm -rf $(BIN_DIR)
	go clean

# Install whisper.cpp CLI
install-whisper:
	@echo "Installing whisper.cpp..."
	@mkdir -p $(WHISPER_DIR)
	@if [ ! -d "$(WHISPER_DIR)/src" ]; then \
		git clone --depth 1 https://github.com/ggml-org/whisper.cpp.git $(WHISPER_DIR)/src; \
	fi
	@cd $(WHISPER_DIR)/src && cmake -B build && cmake --build build --config Release
	@echo "whisper-cli built at $(WHISPER_DIR)/src/build/bin/whisper-cli"
	@echo "Add to PATH or create symlink:"
	@echo "  ln -sf $(WHISPER_DIR)/src/build/bin/whisper-cli /usr/local/bin/whisper-cli"

# Download Whisper model
download-model: $(MODEL_PATH)

$(MODEL_PATH):
	@echo "Downloading Whisper base model..."
	@mkdir -p $(MODEL_DIR)
	@curl -L -o $(MODEL_PATH) \
		"https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.bin"
	@echo "Model downloaded to $(MODEL_PATH)"

# Download smaller model for testing
download-model-tiny:
	@echo "Downloading Whisper tiny model..."
	@mkdir -p $(MODEL_DIR)
	@curl -L -o $(MODEL_DIR)/ggml-tiny.bin \
		"https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-tiny.bin"
	@echo "Tiny model downloaded to $(MODEL_DIR)/ggml-tiny.bin"

# Full setup
setup: install-whisper download-model setup-hooks
	@echo "Setup complete! You can now run 'make build'"

# Run tests
test:
	go test -v ./...

# Run linter
lint:
	golangci-lint run ./...

# Format code
fmt:
	gofmt -w .
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -local github.com/jayzes/memorex -w .; \
	else \
		echo "Note: goimports not installed, run 'go install golang.org/x/tools/cmd/goimports@latest'"; \
	fi

# Setup git hooks
setup-hooks:
	@git config core.hooksPath .githooks
	@chmod +x .githooks/*
	@echo "Git hooks configured to use .githooks/"

# Show help
help:
	@echo "Memorex - Video to Markdown converter"
	@echo ""
	@echo "Targets:"
	@echo "  setup             - Full setup (install whisper.cpp + download model + hooks)"
	@echo "  build             - Build the memorex binary"
	@echo "  install           - Install to GOPATH/bin"
	@echo "  clean             - Remove build artifacts"
	@echo "  install-whisper   - Build whisper.cpp CLI tool"
	@echo "  download-model    - Download Whisper base model (~150MB)"
	@echo "  download-model-tiny - Download Whisper tiny model (~75MB)"
	@echo "  test              - Run tests"
	@echo "  lint              - Run golangci-lint"
	@echo "  fmt               - Format code with gofmt/goimports"
	@echo "  setup-hooks       - Configure git to use .githooks/"
	@echo ""
	@echo "Prerequisites:"
	@echo "  - Go 1.21+"
	@echo "  - FFmpeg"
	@echo "  - CMake and C++ compiler (for whisper.cpp)"
	@echo "  - golangci-lint (for linting)"
