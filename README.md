# Kubernetes Generic Service Broker

![](https://github.com/spjmurray/service-broker/workflows/Build%20and%20Test/badge.svg)

Open Service Broker API driven templating engine for Kubernetes.
The Kubernetes Service Broker conforms to the [Open Service Broker Specification](https://github.com/openservicebrokerapi/servicebroker/blob/v2.13/spec.md) version 2.13.

## End-User Documentation

This is provided as AsciiDoc, organized as an Antora module, and can be found [here](documentation/modules/ROOT/pages/index.adoc).

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

Or for debian DEBs:

```bash
$ make deb -e VERSION=1.0.0 REVISION=beta1
```

## Contributing

### Testing

All code submissions must include sufficient tests to check correctness.
All tests must pass, and do so consistently.
Tests can be run with the following command:

```bash
$ make test
```

You can run individual tests or groups of tests while debugging with the following command:

```bash
$ go test -v -race ./test -run TestConnect -args -logtostderr -v 1
```

Code coverage is run as part of the test command and -- although not enforced, it is watched -- should be checked:

```bask
$ make cover
```

Any code that is added (and not auto-generated) should be covered by testing.
