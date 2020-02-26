VERSION = 0.1.0
GOPATH = $(shell echo $${PWD%/src/*})
SOURCE = $(shell find . -name *.go -type f)
APISRC = $(shell find pkg/apis -name [^z]*.go -type f)
DEPSRC = Gopkg.lock
GENSRC = pkg/revision/revision.go
GENAPI = pkg/generated
BROKER_BIN = build/bin/broker
CRDGEN_BIN = build/bin/crdgen
CRDGEN_FILE = example/crd.yaml

.PHONY: all build dep apigen codegen doc crd container test cover

all: build doc

build: dep ${GENAPI} $(CRDGEN_BIN) $(BROKER_BIN)

dep: vendor

vendor: ${DEPSRC}
	GOPATH=$(GOPATH) dep ensure -vendor-only

${GENAPI}:${APISRC}
	rm -rf pkg/generated
	scripts/codegen/update-generated.sh

$(CRDGEN_BIN): ${SOURCE}
	GOARCH=amd64 CGO_ENABLED=0 go build -o $@ ./cmd/crdgen

$(BROKER_BIN): $(SOURCE)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $@ ./cmd/broker

doc: crd

crd: $(CRDGEN_BIN)

$(CRDGEN_FILE): $(CRDGEN_BIN)
	$@ > $<

container: build
	docker build -f Dockerfile -t couchbase/service-broker:0.0.0 .

test:
	go test -v -cover -coverpkg github.com/couchbase/service-broker/pkg/... -coverprofile=/tmp/cover.out ./test -args -logtostderr

cover:
	go tool cover -html=/tmp/cover.out
