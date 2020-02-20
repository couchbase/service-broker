# Couchbase Kubernetes Service Broker

Open Service Broker API driven templating engine for Kubernetes.

## What are Service Brokers?

Open Service Brokers are an abstraction layer around a service that allow users to create service instances and bind applications to them.
Service instances are abstractions of a service that are controlled by a centralized authority.
This leads to a number of benefits for application developers:

* Service instances are under centralized administrative control
  * Fine graned control over what service instances can be provisioned
    * Security compliance
    * Resource constraining
    * Cost constraining
  * Single pane of glass monitoring and reporting
* Service instances are abstractions
  * No domain specific knowledge necessary to provision and manage services for application developers
  * Consume URIs and credentials
* Enhanced security
  * Elevated privileges to provision service instances are delegated to the broker

## What does the Service Broker do?

The Couchbase Kubernetes Service Broker was originally designed to deploy databases.
Through evolution, it was possible to abstract away all the domain specific knowledge and provide a generic service broker implmenetation, that still supported our original goals.

### Security Model

The Service Broker is designed to be used with the [Kubernetes Service Catalog](https://kubernetes.io/docs/concepts/extend-kubernetes/service-catalog/) which provides Kubernetes native bindings in the form of `ServiceInstance` and `ServiceBinding` resources.
Using Kubernetes RBAC controls, platorm administrators can control precicely what users can provision and where.
This supports:

* Self-service
  * Users can provision both service instances and bind their applications to them
* Shared services
  * Administrators can provision service instances, and users can bind to and consume them 

The Service Broker is flexible enough so that resources created to realize a service instance can be located in the same namespace as the service instance resource, or in a hard coded namespace:

* Namespaced service instances
  * Users may be able to see, and modify, underlying resources, depending on RBAC rules
* Hard-coded namespaced service instances
  * Underlying resources are hidden from users, thus protecting sensitive configuration

The Service Broker is deployed in its own namespace to keep its own configuration and runtime data separate and secured from other users.
Depending on how you wish to configure the Service Broker, it may only require permissions to create resources in its own namespace, or if provisioning resources in other namespaces, cluster wide permissions.

### Templating Engine

The core of the Service Broker is a flexible and generic templating engine.
A service instance or binding is conceptually an ordered list of templates of Kubernetes resources.

Upon creation of an instance the templates are first rendered to apply dynamic configuration from both the environment and the request.
All template rendering operations are carried out in JSON, using [JSON Pointer](https://tools.ietf.org/html/rfc6902) and [JSON Patch](https://tools.ietf.org/html/rfc6902) operations.
Once rendered the resources are then committed to the Kubernetes API.
