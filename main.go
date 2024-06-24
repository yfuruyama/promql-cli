package main

import (
	"flag"
	"log"
	"os"
)

func main() {
	var url, project, headers string

	flag.StringVar(&url, "url", "http://localhost:9090", "The URL for the Prometheus server")
	flag.StringVar(&project, "project", "", "Google Cloud Project ID for Cloud Monitoring")
	flag.StringVar(&headers, "headers", "", "Additional request headers (comma separated)")
	flag.Parse()

	cli, err := NewCLI(url, project, headers, os.Stdin, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}

	exitCode := cli.RunInteractive()
	os.Exit(exitCode)
}
