package main

import (
	"log"

	"github.com/hellofresh/kangal/cmd"
	_ "github.com/hellofresh/kangal/pkg/backends/fake"
	_ "github.com/hellofresh/kangal/pkg/backends/jmeter"
	_ "github.com/hellofresh/kangal/pkg/backends/locust"
)

var version = "0.0.0-dev"

func main() {
	rootCmd := cmd.NewRootCmd(version)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
