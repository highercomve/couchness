# Usage:
# make              # compile all binary
# make clean        # remove ALL binaries and objects
# make test         # run tests
# make deps         # get dependencies
# make build        # compile all binary for all supported platforms
# make build-linux-amd64  # compile linux amd64 binary
# make build-linux-arm  # compile linux arm binary
# make build-linux-arm64  # compile linux arm64 binary
# make build-linux-riscv64  # compile linux arm64 binary
# make optimize     # optimize all binary for all platforms
# make optimize-linux-amd64  # optimize linux amd64 binary
# make optimize-linux-arm  # optimize linux arm binary
# make optimize-linux-arm64  # optimize linux arm64 binary
.PHONY = all clean build build-% optimize optimize-%

BINARY_NAME = couchness

GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
GOGET = $(GOCMD) get
DEBUG ?= 0
OPTIMIZE ?= 1
CGO = CGO_ENABLED=0
export DEBUG OPTIMIZE

ifeq ($(DEBUG), 1)
    BUILD_TYPE = debug
    GCFLAGS = -gcflags="all=-N -l"
    LDFLAGS = -ldflags="-extldflags=-static"
else
    BUILD_TYPE = release
    GCFLAGS = -gcflags="all=-N -l"
    LDFLAGS = -ldflags="-s -w -extldflags=-static"
endif

BUILD_DIR ?= build
export BUILD_DIR

SUPPORTED_PLATFORMS = linux-amd64 linux-arm linux-arm64 linux-riscv64 darwin-amd64 darwin-arm64 windows-amd64 windows-386
SUPPORT_OPTIMIZATION = linux-amd64 linux-arm linux-arm64 windows-amd64 windows-386

all: clean build

# Build the binary for all supported platforms
build:
	$(foreach platform,$(SUPPORTED_PLATFORMS), $(MAKE) build-$(platform);)

# Build the binary for a specific platform
build-%:
	$(eval GOOSARCH := $*)
	$(eval GOOS := $(firstword $(subst -, ,$(GOOSARCH))))
	$(eval GOARCH := $(word 2,$(subst -, ,$(GOOSARCH))))
	$(eval GOARM := $(word 3,$(subst -, ,$(GOOSARCH))))
	$(eval PKG_NAME := $(if $(filter-out windows,$(GOOS)), $(BINARY_NAME), $(BINARY_NAME).exe))
	@echo ""
	@echo "building for $(GOOSARCH) with GOOS=$(GOOS) and GOARCH=$(GOARCH) and GOARM=$(GOARM)"
	@if [ -z "$(OUTPUT_DIR)" ]; then \
		GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) $(CGO) $(GOBUILD) -o $(BUILD_DIR)/$(GOOS)_$(GOARCH)$(GOARM)/$(PKG_NAME) $(GCFLAGS) $(LDFLAGS) -v .; \
	else \
		GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) $(CGO) $(GOBUILD) -o $(OUTPUT_DIR)/$(PKG_NAME) $(GCFLAGS) $(LDFLAGS) -v .; \
	fi
	@if [ "$(OPTIMIZE)" == "1" ]; then make optimize-$(GOOSARCH); fi

# Optimize the binary for all platforms
optimize:
	$(foreach platform,$(SUPPORTED_PLATFORMS), $(MAKE) optimize-$(platform);)

# Optimize the binary for a specific platform
optimize-%:
	$(eval GOOSARCH := $*)
	$(eval GOOS := $(firstword $(subst -, ,$(GOOSARCH))))
	$(eval GOARCH := $(word 2,$(subst -, ,$(GOOSARCH))))
	$(eval GOARM := $(word 3,$(subst -, ,$(GOOSARCH))))
	$(eval PKG_NAME := $(if $(filter-out windows,$(GOOS)), $(BINARY_NAME), $(BINARY_NAME).exe))
	@echo "optimizing for $(GOOSARCH) with GOOS=$(GOOS) and GOARCH=$(GOARCH) and GOARM=$(GOARM)"
	@if [ -z "$(OUTPUT_DIR)" ]; then \
		if [[ " $(SUPPORT_OPTIMIZATION) " =~ " $(GOOSARCH) " ]]; then \
			upx $(BUILD_DIR)/$(GOOS)_$(GOARCH)$(GOARM)/$(PKG_NAME); \
		fi \
	else \
		if [[ " $(SUPPORT_OPTIMIZATION) " =~ " $(GOOSARCH) " ]]; then \
			upx $(OUTPUT_DIR)/$(PKG_NAME); \
		fi \
	fi

# Clean target
clean:
	rm -rf build/

# Test target
test:
	$(GOTEST) -v ./... && ./run_test.sh

# Get dependencies
deps:
	$(GOGET) -d ./...
