module github.com/kindlyops/vbs

go 1.15

require (
	github.com/aws/aws-sdk-go v1.36.28
	// If changing rules_go version, remember to change version in WORKSPACE also
	github.com/bazelbuild/rules_go v0.28.0
	github.com/hypebeast/go-osc v0.0.0-20200115085105-85fee7fed692
	github.com/kennygrant/sanitize v1.2.4
	github.com/mattn/go-isatty v0.0.3
	github.com/mitchellh/go-homedir v1.1.0
	github.com/rs/zerolog v1.22.0
	github.com/spf13/cobra v0.0.7
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.5.1 // indirect
)
