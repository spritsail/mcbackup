package main

import (
	"github.com/jessevdk/go-flags"
	"github.com/sirupsen/logrus"
	"github.com/spritsail/mcbackup/config"
	"github.com/spritsail/mcbackup/mcbackup"
	providers "github.com/spritsail/mcbackup/provider/load"
	"github.com/x-cray/logrus-prefixed-formatter"
	"os"
)

var Version string

func init() {
	logrus.SetFormatter(&prefixed.TextFormatter{
		FullTimestamp: true,
	})
	logrus.SetLevel(logrus.DebugLevel)
}

func main() {
	var opts config.GlobalOpts

	log := logrus.WithField("prefix", "main")
	log.Printf("mcbackup, version %s", Version)

	// Parse global commandline options, ignoring anything unknown
	// so that they can be re-parsed by the provider.
	parser := flags.NewParser(&opts, flags.IgnoreUnknown|flags.HelpFlag)
	parser.Name = "mcbackup"

	remain, err := parser.ParseArgs(os.Args)
	if err != nil {
		log.Error(err)
		parser.WriteHelp(os.Stdout)
		os.Exit(1)
	}

	// Find the provider named by argument/environment variable
	providerInit := providers.Find(opts.Provider)
	if providerInit == nil {
		log.Error("No such provider found with name '%s'", opts.Provider)
		os.Exit(1)
	}

	// Attempt to initialise the provider with the remaining arguments
	prov, remain, err := providerInit(remain)
	if err != nil {
		log.Error("Failed to create provider")
		log.Error(err)
		os.Exit(1)
	}

	// Any extraneous values not consumed by the provider WILL throw and error here
	// Create new parser to parse remaining options, catching unknown arguments
	var empty struct{}
	_, err = flags.NewParser(&empty, 0).ParseArgs(remain)
	if err != nil {
		log.Error(err)
		parser.WriteHelp(os.Stdout)
		os.Exit(1)
	}

	// Start the backup process
	err = mcbackup.New(prov, &opts).Run()

	if err != nil {
		log.Fatal(err)
	}
}
