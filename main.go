package main

import (
	"github.com/jessevdk/go-flags"
	"github.com/seeruk/minecraft-rcon/rcon"
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
	// Required to prevent default action of single backup
	parser.SubcommandsOptional = true

	remain, err := parser.ParseArgs(os.Args[1:])
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

	log.Debug("creating client")
	client, err := rcon.NewClient(
		opts.Host,
		int(opts.Port),
		opts.Password,
	)
	if err != nil {
		logrus.WithField("prefix", "rcon").Error(err)
		log.Fatal("error creating client")
	}

	mcb := mcbackup.New(prov, client, &opts)
	switch parser.Active {
	case nil:
		log.Info("running a single backup")
		mcb.RunOnce()
		break
	default:
		cmd := parser.Active.Name
		switch cmd {
		case "cron":
			mcb.Cron()
		default:
			log.Fatalf("unknown command '%s'", cmd)
		}
	}
}
