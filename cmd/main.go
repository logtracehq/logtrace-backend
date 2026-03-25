package main

import (
	"log"
	"os"

	"gitlab.com/logtrace/logtrace/cmd/cli"
)

func main() {
	os.Setenv("TZ", "")

	if err := cli.Execute(); err != nil {
		log.Fatal(err)
	}
}
