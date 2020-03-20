package version

var (
	// Application is the application name.
	Application string

	// Version is the release version, this is set during the build process.
	Version string

	// GitCommit is the git revision, this is set during the build process.
	GitCommit string
)
