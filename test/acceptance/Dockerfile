FROM scratch

# Copy the main test binary.
COPY build/bin/acceptance /usr/local/bin/

# Examples are our acceptance tests, they test the Kubernetes API and a
# full end-to-end workflow.
COPY examples /usr/local/share/couchbase-service-broker/examples

# CRDs are generated and need to be included in the image in order to
# be installed.
COPY crds /usr/local/share/couchbase-service-broker/crds

ENTRYPOINT ["/usr/local/bin/acceptance"]
