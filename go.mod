module github.com/daedaleanai/dbt

go 1.13

require (
	dbt v0.0.0-00010101000000-000000000000
	github.com/briandowns/spinner v1.12.0
	github.com/go-git/go-git v4.7.0+incompatible
	github.com/go-git/go-git/v5 v5.2.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/sirupsen/logrus v1.2.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.1
	gopkg.in/yaml.v2 v2.2.8
)

replace dbt => ./
