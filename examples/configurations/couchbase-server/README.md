# Couchbase Server Example Configuration

This example creates a Couchbase Server cluster, an in-memory NoSQL database.
It is deployed and managed by the [Couchbase Autonomous Operator](https://www.couchbase.com/products/cloud/kubernetes).

## Licensing

Use of the Couchbase Autonomous Operator is governed by the [Couchbase License Agreement Version 7](https://www.couchbase.com/LA11122019).

## Prerequisites

By default, the Service Broker is deployed namespace scoped.
This is by design so that users don't get obsessed by cluster scoped roles.
This does, however, mean that it cannot deploy the necessary components for a fully functional, ground up deployment.
You will need to deploy the dynamic admission controller separately before provisioning service instances with this example.
