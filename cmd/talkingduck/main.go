package main

import (
	"flag"
	"log"
	"os"

	"talkingduck/cli"
)

func main() {
	log.SetFlags(0)
	opts := cli.RegisterFlags(flag.CommandLine)
	flag.Parse()
	os.Exit(cli.Run(opts, flag.Args()))
}
