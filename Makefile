# These are controlled by the CI/CD system when an official build is produced.
# Development builds are distinguished by git commit.
APPLICATION = couchbase-service-broker
VERSION = 0.0.0
IMAGE = couchbase/service-broker

# These are specific to the build system.
BUILD_DIR = build
EXAMPLE_DIR = examples
GENERATED_DIR = generated
ARCHIVE_DIR = archives
CRD_DIR = crds
WROKSPACE = $(PWD)

COMMIT = $(shell git rev-parse HEAD)
GOPATH = $(shell echo $${PWD%/src/*})
SOURCE = $(shell find . -name *.go -type f)
APISRC = $(shell find pkg/apis -name [^z]*.go -type f)
EXAMPLES = $(shell find $(EXAMPLE_DIR))
DEPSRC = go.mod
GENSRC = pkg/revision/revision.go
BROKER_BIN = $(BUILD_DIR)/bin/broker
CRDGEN_FILE = $(CRD_DIR)/broker.couchbase.com_couchbaseservicebrokerconfigs.yaml
COVER_FILE = /tmp/cover.out
CODEGEN = vendor/k8s.io/code-generator
IMPORTPATH = github.com/couchbase/service-broker
ARCHIVE_BASE = $(APPLICATION)-$(VERSION)
ARCHIVE_TGZ = $(ARCHIVE_BASE).tar.gz
ARCHIVE_ZIP = $(ARCHIVE_BASE).zip
STATIC_FILES = LICENSE README.md Dockerfile

.PHONY: all build dep apigen doc crd container test unit lint cover archive archive-tgz archive-zip

all: build doc

build: $(BROKER_BIN)

$(GENERATED_DIR): $(CODEGEN) $(APISRC)
	rm -rf $(GENERATED_DIR)
	GOPATH=$(HOME) ./vendor/k8s.io/code-generator/generate-groups.sh all github.com/couchbase/service-broker/generated github.com/couchbase/service-broker/pkg/apis broker.couchbase.com:v1alpha1 --go-header-file hack/boilerplate.go.txt --output-base ../../..

$(CODEGEN):
	git clone -b kubernetes-1.13.4 https://github.com/kubernetes/code-generator $(CODEGEN)

$(BROKER_BIN): $(GENERATED_DIR) $(SOURCE)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-X $(IMPORTPATH)/pkg/version.Application=$(APPLICATION) -X $(IMPORTPATH)/pkg/version.Version=$(VERSION) -X $(IMPORTPATH)/pkg/version.GitCommit=$(COMMIT)" -o $@ ./cmd/broker

doc: crd

crd: $(CRDGEN_FILE)

$(CRD_DIR):
	mkdir -p $@

$(CRDGEN_FILE): $(CRD_DIR) $(APISRC)
	go run sigs.k8s.io/controller-tools/cmd/controller-gen crd paths=./pkg/apis/... output:dir=$(CRD_DIR)

container: build
	docker build -f Dockerfile -t $(IMAGE):$(VERSION) .

test: lint unit

lint: ${GENERATED_DIR}
	go run github.com/golangci/golangci-lint/cmd/golangci-lint run

unit: ${GENERATED_DIR}
	go test -v -race -cover -coverpkg github.com/couchbase/service-broker/pkg/... -coverprofile=$(COVER_FILE) ./test

cover:
	go tool cover -html=$(COVER_FILE)

archive: archive-tgz archive-zip

archive-tgz: $(ARCHIVE_TGZ)

$(ARCHIVE_TGZ): $(ARCHIVE_DIR)
	tar -czf $(ARCHIVE_TGZ) -C $(ARCHIVE_DIR) $(ARCHIVE_BASE)

archive-zip: $(ARCHIVE_ZIP)

$(ARCHIVE_ZIP): $(ARCHIVE_DIR)
	cd $(ARCHIVE_DIR); zip -r $(ARCHIVE_ZIP) $(ARCHIVE_BASE)
	mv $(ARCHIVE_DIR)/$(ARCHIVE_ZIP) .

$(ARCHIVE_DIR): $(STATIC_FILES) $(EXAMPLES) $(CRDGEN_FILE) $(BROKER_BIN)
	rm -rf $(ARCHIVE_DIR)
	mkdir -p $(ARCHIVE_DIR)/$(ARCHIVE_BASE)
	cp -a $(EXAMPLE_DIR) $(STATIC_FILES) $(ARCHIVE_DIR)/$(ARCHIVE_BASE)
	find $(ARCHIVE_DIR)/$(ARCHIVE_BASE) -type f -exec sed -i "s/0\.0\.0/$(VERSION)/g" {} \;
	cp -a $(CRDGEN_FILE) $(ARCHIVE_DIR)/$(ARCHIVE_BASE)/$(EXAMPLE_DIR)
	cp -a $(BUILD_DIR) $(ARCHIVE_DIR)/$(ARCHIVE_BASE)
