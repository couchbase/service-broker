package version

const (
	// Application is the application name.
	Application = "couchbase-service-broker"
)

var (
	// Version is the release version, this is set during the build process.
	Version string

	// GitCommit is the git revision, this is set during the build process.
	GitCommit string
)
