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
	mode, err := opts.Mode("tui")
	if err != nil {
		log.Print(err)
		os.Exit(2)
	}
	switch mode {
	case "gui":
		if err := cli.LaunchGUI(os.Args[1:]); err != nil {
			log.Print(err)
			os.Exit(1)
		}
	case "tui":
		os.Exit(cli.RunTUI(opts, flag.Args()))
	default:
		os.Exit(cli.Run(opts, flag.Args()))
	}
}
