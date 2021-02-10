module github.com/daedaleanai/dbt

go 1.13

require (
	dbt v0.0.0
	github.com/briandowns/spinner v1.12.0
	github.com/daedaleanai/cobra v1.1.1-2
	github.com/go-git/go-git/v5 v5.2.0
	gopkg.in/yaml.v2 v2.4.0
)

replace dbt v0.0.0 => ./
