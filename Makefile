VER ?= $(VERSION)
LDFLAGS = -X "main.BuildTimestamp=$(shell date -u '+%Y-%m-%d %H:%M:%S')" \
          -X "main.VERSION=$(VER)" \
          -X "main.goVersion=$(shell go version | sed -r 's/go version go(.*)\ .*/\1/')"

GO := GO111MODULE=on CGO_ENABLED=0 go
GOLANGCI_LINT_VERSION = v1.61.0

GOOS_LIST := linux windows
GOARCH_LIST := amd64 arm64 arm 386

.PHONY: build
build:
	@mkdir -p bin dist
	@for goos in $(GOOS_LIST); do \
		if [ "$$goos" = "windows" ]; then \
			ext=".exe"; \
		else \
			ext=""; \
		fi; \
		for goarch in $(GOARCH_LIST); do \
			echo "Building for $$goos-$$goarch..."; \
			outfile="bin/BPB-Wizard-$$goos-$$goarch$$ext"; \
			GOOS=$$goos GOARCH=$$goarch $(GO) build -ldflags '$(LDFLAGS)' -o $$outfile || { echo "Not supported, Skipping $$goos-$$goarch"; continue; }; \
			zipfile="dist/BPB-Wizard-$$goos-$$goarch.zip"; \
			echo "Zipping $$outfile -> $$zipfile"; \
			zip -j -q $$zipfile $$outfile; \
		done; \
	done
