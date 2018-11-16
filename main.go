package main

import (
	"os"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/sirupsen/logrus"
	"github.com/spritsail/mcbackup/config"
	"github.com/spritsail/mcbackup/mcbackup"
	"github.com/spritsail/mcbackup/provider"
	"github.com/x-cray/logrus-prefixed-formatter"
)

// Version constant set with -ldflags '-X main.Version=xxx'
var Version string

func init() {
	logrus.SetFormatter(&prefixed.TextFormatter{
		FullTimestamp: true,
	})
}

func main() {
	var opts config.Options

	log := logrus.WithField("prefix", "main")
	log.Printf("mcbackup, version %s", Version)

	// Parse global commandline options, ignoring anything unknown
	// so that they can be re-parsed by the provider.
	parser := flags.NewParser(&opts, flags.IgnoreUnknown|flags.HelpFlag)
	parser.Name = "mcbackup"
	parser.SubcommandsOptional = true

	remain, err := parser.ParseArgs(os.Args[1:])
	if err != nil {
		// Handle 'no command specified' scenario by defaulting to 'once'
		if e, ok := err.(*flags.Error); ok &&
			e.Type == flags.ErrCommandRequired {
			// This isn't actually an error.
			// If no command is provided, just default to the 'once' command
			parser.Active = parser.Find("once")
		} else {
			if e, ok := err.(*flags.Error); ok &&
				e.Type != flags.ErrHelp {
				log.Error(err)
			}
			parser.WriteHelp(os.Stderr)
			os.Exit(1)
		}
	}

	lvl, err := logrus.ParseLevel(opts.LogLevel)
	if err != nil {
		log.WithError(err).
			Errorf("failed to set log level '%s'", opts.LogLevel)
	}
	logrus.SetLevel(lvl)

	// Find the provider named by argument/environment variable
	providerInit := provider.Find(opts.Provider)
	if providerInit == nil {
		log.Errorf("No such provider found with name '%s'", opts.Provider)
		os.Exit(1)
	}

	// Attempt to initialise the provider with the remaining arguments
	prov, remain, err := providerInit(remain, &opts)
	if err != nil {
		log.WithField("name", opts.Provider).
			WithError(err).
			Error("Failed to create provider")
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
	client, err := mcbackup.NewClient(&opts)
	if err != nil {
		logrus.
			WithField("prefix", "rcon").
			WithError(err).
			Fatal("error creating client")
	}
	logrus.WithField("prefix", "rcon").
		Info("client connection successful")

	mcb := mcbackup.New(prov, client, &opts)
	command := parser.Active
	if command == nil {
		command = parser.Find("once")
	}
	switch command.Name {
	case "cron":
		mcb.Cron()
		break
	case "prune":
		err = mcb.Prune(time.Now())
		break
	default:
	case "once":
		log.Info("running a single backup")
		err = mcb.RunOnce(time.Now())
		break
	}

	if err != nil {
		log.WithError(err).
			Fatalf(":(")
	}
}
