#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

go install ./vendor/k8s.io/code-generator/cmd/...

./scripts/codegen/codegen.sh \
  "all" \
  "github.com/couchbase/service-broker/pkg/generated" \
  "github.com/couchbase/service-broker/pkg/apis" \
  "broker.couchbase.com:v1" \
  --go-header-file "./scripts/codegen/boilerplate.go.txt" \
  $@
