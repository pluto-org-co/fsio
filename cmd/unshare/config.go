package main

type (
	Drive struct {
		AccountFile string `yaml:"account-file"`
		Subject     string `yaml:"subject"`
	}
	Config struct {
		Drive Drive `yaml:"drive"`
	}
)
