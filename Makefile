VERSION = 0.1.0
GOPATH = $(shell echo $${PWD%/src/*})
SOURCE = $(shell find . -name *.go -type f)
APISRC = $(shell find pkg/apis -name [^z]*.go -type f)
DEPSRC = Gopkg.lock
GENSRC = pkg/revision/revision.go
GENAPI = pkg/generated
BROKER_BIN = build/bin/broker
CRDGEN_FILE = example/broker.couchbase.com_couchbaseservicebrokerconfigs.yaml
COVER_FILE=/tmp/cover.out

.PHONY: all build dep apigen codegen doc crd container test cover

all: build doc

build: dep ${GENAPI} $(CRDGEN_BIN) $(BROKER_BIN)

dep: vendor

vendor: $(DEPSRC)
	GOPATH=$(GOPATH) dep ensure -vendor-only

$(GENAPI): $(APISRC)
	rm -rf pkg/generated
	scripts/codegen/update-generated.sh

$(BROKER_BIN): $(SOURCE)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $@ ./cmd/broker

doc: crd

crd: $(CRDGEN_FILE)

$(CRDGEN_FILE): $(APISRC)
	go run sigs.k8s.io/controller-tools/cmd/controller-gen crd paths=./pkg/apis/... output:dir=./example

container: build
	docker build -f Dockerfile -t couchbase/service-broker:0.0.0 .

test:
	go vet ./...
	go test -v -race -cover -coverpkg github.com/couchbase/service-broker/pkg/... -coverprofile=$(COVER_FILE) ./test -args -logtostderr

cover:
	go tool cover -html=$(COVER_FILE)
