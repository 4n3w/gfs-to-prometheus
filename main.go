package main

import (
	"log"
	"os"

	"github.com/4n3w/gfs-to-prometheus/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}