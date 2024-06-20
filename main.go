package main

import (
	"flag"
	"log"
	"os"
)

func main() {
	var url string

	flag.StringVar(&url, "url", "http://localhost:9090", "The URL for the Prometheus server")
	flag.Parse()

	cli, err := NewCli(url, os.Stdin, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}

	exitCode := cli.RunInteractive()
	os.Exit(exitCode)
}
