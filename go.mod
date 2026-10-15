module github.com/echocat/cli4g

go 1.27.0

require (
	github.com/stretchr/testify v1.12.1
	golang.org/x/mod v0.42.0
)

require go.yaml.in/yaml/v3 v3.0.5 // indirect

ignore (
	./.github
	./.idea
	./.run
	./_benchmarks
	./_examples
	./mise-tasks
)
