VERSION = 0.1.0
GIT_COMMIT = $(shell git rev-parse HEAD)
GOPATH = $(shell echo $${PWD%/src/*})
SOURCE = $(shell find . -name *.go -type f)
APISRC = $(shell find pkg/apis -name [^z]*.go -type f)
DEPSRC = go.mod
GENSRC = pkg/revision/revision.go
GENAPI = generated
BROKER_BIN = build/bin/broker
CRDGEN_FILE = example/broker.couchbase.com_couchbaseservicebrokerconfigs.yaml
COVER_FILE=/tmp/cover.out
CODEGEN = vendor/k8s.io/code-generator
IMPORTPATH=github.com/couchbase/service-broker

.PHONY: all build dep apigen doc crd container test cover

all: build doc

build: ${GENAPI} $(BROKER_BIN)

$(GENAPI): $(CODEGEN) $(APISRC)
	rm -rf $(GENAPI)
	GOPATH=$(HOME) ./vendor/k8s.io/code-generator/generate-groups.sh all github.com/couchbase/service-broker/generated github.com/couchbase/service-broker/pkg/apis broker.couchbase.com:v1 --go-header-file hack/boilerplate.go.txt --output-base ../../..

$(CODEGEN):
	git clone -b kubernetes-1.13.4 https://github.com/kubernetes/code-generator $(CODEGEN)

$(BROKER_BIN): $(SOURCE)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-X $(IMPORTPATH)/pkg/version.Version=$(VERSION) -X $(IMPORTPATH)/pkg/version.GitCommit=$(GIT_COMMIT)" -o $@ ./cmd/broker

doc: crd

crd: $(CRDGEN_FILE)

$(CRDGEN_FILE): $(APISRC)
	go run sigs.k8s.io/controller-tools/cmd/controller-gen crd paths=./pkg/apis/... output:dir=./example

container: build
	docker build -f Dockerfile -t couchbase/service-broker:$(VERSION) .

test: ${GENAPI}
	go run github.com/golangci/golangci-lint/cmd/golangci-lint run
	go test -v -race -cover -coverpkg github.com/couchbase/service-broker/pkg/... -coverprofile=$(COVER_FILE) ./test

cover:
	go tool cover -html=$(COVER_FILE)
