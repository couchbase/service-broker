Name: couchbase-service-broker
Version: 0.0.0
Release: 99999
License: Apache
Summary: Kubernetes Generic Service Broker
Group: System/Daemons
URL: https://github.com/couchbase/service-broker
Source: couchbase-service-broker-0.0.0.tar.gz

%description
Kubernetes daemon that implements the Open Service Broker API.  It is powered
by a flexible templating engine that allows any Kubernetes resource or set of
resources to be configured and deployed with a single API call.  The broker
administrator is responsible for controlling what is deployed, therefore the
broker provides an abstraction layer around an application - hiding domain
specific complexity - and allows delegation of required privilege escalation
to the broker and not an end user.

%prep
%setup

%build
make VERSION=%{version}-%{release}

%install
make install PREFIX=%{buildroot}/usr/share VERSION=%{version}-%{release}

%files
/
