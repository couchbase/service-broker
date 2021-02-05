# Kubernetes Generic Service Broker Contribution Guide

How to develop the Service Broker and make contributions.

## Development

### Building the Container Image from Source

To build a container from source you can use the following command:

```bash
$ make container
```

### Building a Release from Source

To build a release from source:

```bash
$ make archive -e VERSION=1.0.0 REVISION=beta1 DESTDIR=/tmp/archive
```

Or for Red Hat RPMs:

```bash
$ make rpm -e VERSION=1.0.0 REVISION=beta1
```

Or for Debian DEBs:

```bash
$ make deb -e VERSION=1.0.0 REVISION=beta1
```

## Contributing

### Generated Code

Resource CRDs and Kubernetes clients are generated in response to modifications in the files they depend upon.
These files must be checked into any commits affecting these files.
A client should be able to clone an use the APIs and clients without any external tooling.
Likewise CRDs are linked to from the documentation and must be kept up to date.

### Testing Code Submissions

All code submissions must include sufficient tests to check correctness.
All tests must pass, and do so consistently.
These tests are an amalgamation of unit and integration testing.

Tests can be run with the following command:

```bash
$ make test
```

You can run individual tests or groups of tests while debugging with the following command:

```bash
$ go test -v -race ./test/unit -run TestConnect -args -logtostderr -v 1
```

Code coverage is run as part of the test command and -- although not enforced, it is watched -- should be checked:

```bask
$ make cover
```

Any code that is added (and not auto-generated) should be covered by testing.

See below for additional testing of examples.

### Testing Example Submissions

Examples define sample configurations for a specific applications.
Acceptance tests provide end-to-end testing of the service broker and for acceptance and system testing.
These tests also aid in testing the installation documentation and ensure all configuration works.
Acceptance tests are not automated as part of continuous integration, but you will be expected to test and confirm your submissions work.

Acceptance testing is done with kind (Kubernetes in Docker), you must first install the Kubernetes Service Catalog.

Acceptance tests can then be run with:

```bash
$ make acceptance
```

The acceptance tests will first install all CRDs with the current versions.
Then for every configuration defined it will install the configuration (testing CRD validation).
Next it will install the service broker, testing that the configuration validity condition is valid and the service broker is ready.
Finally it will create the service instance associated with a configuration and optionally a service binding, before doing a controlled tear-down in reverse.

The obvious rule here is that an example configuration must be able to provision a service instance without any external dependencies.

The important files are:

#### examples/broker.yaml

This contains the service broker service, deployment, rolebinding and service account.
When combined with a configuration it should yield a working service broker service.

#### examples/clusterservicebroker.yaml

This is used to register the service broker with the service catalog.

#### examples/configurations/my-configuration

Every configuration has its own directory, _my-configuration_ in this case.
The acceptance tests will dynamically create tests for each configuration.

#### examples/configurations/my-configuration/prerequisites.yaml

Configurations may supply a set of prerequisite resources, such as CRDs etc. that the service broker cannot configure.

#### examples/configurations/my-configuration/broker.yaml

Every configuration must have a service broker configuration, and role that allows the configuration to create and delete a service instance (optionally a service binding).

#### examples/configurations/my-configuration/serviceinstance.yaml

Every configuration must have a service instance definition to tests that service instance creation and deletion function correctly.

#### examples/configurations/my-configuration/servicebinding.yaml

A configuration may have a service binding definition, this will test that a service instance can be bound to.

## Documentation

If you write something new, don't assume users will magically be able to know about it, document it!
The CI system will do a rudimentary spell check and tell you how to fix it, however you can save yourself the effort and run:

```bash
$ make docs
```

### Documentation Rules

* One line per sentence - easier to diff.
* No complex words - English may not be someone's first language.
* Be clear and concise, do not go off on a tangent - don't confuse the reader.
* Don't repeat yourself - keep DRY.
* Be version agnostic - avoid having to continually update things.
* Use [images](diagrams.net) - _a picture paints a thousand words!_
* Use source code callouts - draw attention to interesting code snippets.
* Use `xref`, not `link` within the module - xrefs are error checked.
* Don't use includes, GitHub doesn't allow this - we've only been waiting for 3 years...
* All admonitions and images need a title - quick summary whether soneone needs to read this.
* If in doubt, follow the existing style.
