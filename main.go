package main

import (
	"flag"
	"log"
	"os"
)

func main() {
	var url, project string

	flag.StringVar(&url, "url", "http://localhost:9090", "The URL for the Prometheus server")
	flag.StringVar(&project, "project", "", "Google Cloud Project ID for Cloud Monitoring")
	flag.Parse()

	cli, err := NewCli(url, project, os.Stdin, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}

	exitCode := cli.RunInteractive()
	os.Exit(exitCode)
}
