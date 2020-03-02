VERSION = 0.1.0
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

.PHONY: all build dep apigen doc crd container test cover

all: build doc

build: ${GENAPI} $(BROKER_BIN)

$(GENAPI): $(CODEGEN) $(APISRC)
	rm -rf $(GENAPI)
	GOPATH=$(HOME) ./vendor/k8s.io/code-generator/generate-groups.sh all github.com/couchbase/service-broker/generated github.com/couchbase/service-broker/pkg/apis broker.couchbase.com:v1 --go-header-file hack/boilerplate.go.txt --output-base ../../..

$(CODEGEN):
	git clone -b kubernetes-1.13.4 https://github.com/kubernetes/code-generator $(CODEGEN)

$(BROKER_BIN): $(SOURCE)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $@ ./cmd/broker

doc: crd

crd: $(CRDGEN_FILE)

$(CRDGEN_FILE): $(APISRC)
	go run sigs.k8s.io/controller-tools/cmd/controller-gen crd paths=./pkg/apis/... output:dir=./example

container: build
	docker build -f Dockerfile -t couchbase/service-broker:0.0.0 .

test: ${GENAPI}
	go vet ./...
	go test -v -race -cover -coverpkg github.com/couchbase/service-broker/pkg/... -coverprofile=$(COVER_FILE) ./test

cover:
	go tool cover -html=$(COVER_FILE)
