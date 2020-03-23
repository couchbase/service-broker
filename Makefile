################################################################################
# Variables
################################################################################

# These are controlled by the CI/CD system when an official build is produced.
# Development builds are distinguished by git commit.
APPLICATION = couchbase-service-broker
VERSION = 0.0.0
IMPORT_PATH = github.com/couchbase/service-broker
DOCKER_IMAGE = couchbase/service-broker
PREFIX = /usr

################################################################################
# Constants
################################################################################

# These are specific to the build system.
BUILD_DIR = build
EXAMPLE_DIR = examples
GENERATED_DIR = generated
CRD_DIR = crds
COMMIT = $(shell git rev-parse HEAD)
SOURCE = $(shell find . -name *.go -type f)
APISRC = $(shell find pkg/apis -name [^z]*.go -type f)
EXAMPLES = $(shell find $(EXAMPLE_DIR))
DEPSRC = go.mod
GENSRC = pkg/revision/revision.go
BROKER_BIN = $(BUILD_DIR)/bin/broker
COVER_FILE = /tmp/cover.out
ARCHIVE_BASE = $(APPLICATION)-$(VERSION)
ARCHIVE_DIR = $(BUILD_DIR)/$(ARCHIVE_BASE)
ARCHIVE_TGZ = $(ARCHIVE_BASE).tar.gz
ARCHIVE_ZIP = $(ARCHIVE_BASE).zip
STATIC_FILES = LICENSE README.md Dockerfile
GENAPIBASE = github.com/couchbase/service-broker/pkg/apis
GENAPIS = $(GENAPIBASE)/broker.couchbase.com/v1alpha1
GENARGS = --go-header-file hack/boilerplate.go.txt --output-base ../../..
GENCLIENTNAME = servicebroker
GENCLIENTS = $(IMPORT_PATH)/$(GENERATED_DIR)/clientset
GENLISTERS = $(IMPORT_PATH)/$(GENERATED_DIR)/listers
GENINFORMERS = $(IMPORT_PATH)/$(GENERATED_DIR)/informers

################################################################################
# Top level make targets.
################################################################################

# These phony targets do not refer to actual files and are intended to be
# invoked by the end user.
.PHONY: all build crd container test unit lint cover archive archive-tgz archive-zip install

# Main build target, makes the binary and CRD.
all: build crd

# Build the main binary.
build: $(BROKER_BIN)

# Build a container image.
container: build
	docker build -f Dockerfile -t $(DOCKER_IMAGE):$(VERSION) .

# Build the CRDs.
crd: $(CRD_DIR)

# Main test target, run linter and all tests.
test: lint unit

# Render code coverage (after running the test target) and display it in a browser.
cover:
	go tool cover -html=$(COVER_FILE)

# The linter must pass for all code submissions, it is controlled by .golangci.yml.
lint: ${GENERATED_DIR}
	go run github.com/golangci/golangci-lint/cmd/golangci-lint run

# The unit tests must pass for all code submissions, additionally code
# coverage should be checked to ensure code submissions actually work.
unit: ${GENERATED_DIR}
	go test -v -race -cover -coverpkg github.com/couchbase/service-broker/pkg/... -coverprofile=$(COVER_FILE) ./test

# Main archival target, creates TGZ and ZIP artifacts.
archive: archive-tgz archive-zip

# Create a TGZ release artifact.
archive-tgz: $(ARCHIVE_TGZ)

# Create a ZIP release artifact.
archive-zip: $(ARCHIVE_ZIP)

# Clean all generated code and artifacts.
clean:
	rm -rf $(BUILD_DIR) $(CRD_DIR) $(ARCHIVE_DIR) $(ARCHIVE_TGZ) $(ARCHIVE_ZIP)

# Install copies from the processed install directory to the specified
# prefix.  Used for DEB and RPM builds.
install: $(ARCHIVE_DIR)
	cp -a $(ARCHIVE_DIR) $(PREFIX)/share

################################################################################
# Make rules
################################################################################

# Generated code depends upon API sources. The code generator still requires a
# GOPATH style install hence the hacks with the output base.  This may get fixed
# in a later release.
$(GENERATED_DIR): $(APISRC)
	rm -rf $(GENERATED_DIR)
	go run k8s.io/code-generator/cmd/deepcopy-gen --input-dirs $(GENAPIS) -O zz_generated.deepcopy --bounding-dirs $(GENAPIBASE) $(GENARGS)
	go run k8s.io/code-generator/cmd/client-gen --clientset-name $(GENCLIENTNAME) --input-base "" --input $(GENAPIS) --output-package $(GENCLIENTS) $(GENARGS)
	go run k8s.io/code-generator/cmd/lister-gen --input-dirs $(GENAPIS) --output-package $(GENLISTERS) $(GENARGS)
	go run k8s.io/code-generator/cmd/informer-gen --input-dirs $(GENAPIS) --versioned-clientset-package $(GENCLIENTS)/$(GENCLIENTNAME) --listers-package $(GENLISTERS) --output-package $(GENINFORMERS) $(GENARGS)

# The main broker binary depends on generated code and all source.
# This should be the contents of pkg/ and the main file for correctness.
$(BROKER_BIN): $(GENERATED_DIR) $(SOURCE)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-X $(IMPORT_PATH)/pkg/version.Application=$(APPLICATION) -X $(IMPORT_PATH)/pkg/version.Version=$(VERSION) -X $(IMPORT_PATH)/pkg/version.GitCommit=$(COMMIT)" -o $@ ./cmd/broker

# The CRDs are auto generated and depend on the API source only.
$(CRD_DIR): $(APISRC)
	rm -rf $@
	mkdir -p $@
	go run sigs.k8s.io/controller-tools/cmd/controller-gen crd paths=./pkg/apis/... output:dir=$(CRD_DIR)

# The TGZ archive relies on the archive directory.
$(ARCHIVE_TGZ): $(ARCHIVE_DIR)
	tar -czf $@ -C $(BUILD_DIR) $(ARCHIVE_BASE)

# The ZIP archive relies on the archive directory.
$(ARCHIVE_ZIP): $(ARCHIVE_DIR)
	cd $(BUILD_DIR); zip -r $@ $(ARCHIVE_BASE)
	mv $(BUILD_DIR)/$@ .

# The archive directory is used to generate release packages (tar.gz or zip).
# Static resources are copied over first and processed to replace the magic
# 0.0.0 with the version supplied by the environment.  This affects docs and
# things like makefiles.  Finally the binaries are copied in.
# This is a quick hack, and it should use an install target that can be used
# for both RPM and DEB generation in future.
$(ARCHIVE_DIR): $(STATIC_FILES) $(EXAMPLES) $(CRD_DIR) $(BROKER_BIN)
	rm -rf $@
	mkdir -p $@
	cp -a $(EXAMPLE_DIR) $(STATIC_FILES) $@
	find $@ -type f -exec sed -i "s/0\.0\.0/$(VERSION)/g" {} \;
	cp -a $(CRD_DIR)/* $@/$(EXAMPLE_DIR)
	cp -a $(BUILD_DIR)/bin $@
