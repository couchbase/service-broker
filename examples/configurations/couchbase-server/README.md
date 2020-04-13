# Couchbase Server Example Configuration

This example creates a Couchbase Server cluster.
It is deployed and managed by the Couchbase Autonomous Operator.

## Prerequisites

By default, the Service Broker is deployed namespace scoped.
This is by design so that users don't get obsessed by cluster scoped roles.
This does, however, mean that it cannot deploy the necessary components for a fully functional, ground up deployment.
You will need to deploy the dynamic admission controller separately before provisioning service instances with this example.
