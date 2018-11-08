package main

import (
	"github.com/jessevdk/go-flags"
	"github.com/sirupsen/logrus"
	"github.com/spritsail/mcbackup/config"
	"github.com/spritsail/mcbackup/provider"
	providers "github.com/spritsail/mcbackup/provider/load"
	"github.com/spritsail/mcbackup/rcon"
	"github.com/x-cray/logrus-prefixed-formatter"
	"os"
)

func init() {
	logrus.SetFormatter(&prefixed.TextFormatter{
		FullTimestamp: true,
	})
	logrus.SetLevel(logrus.DebugLevel)
	logrus.WithField("prefix", "main")
}

func main() {
	var opts config.GlobalOpts

	log := logrus.WithField("prefix", "main")

	// Parse global commandline options, ignoring anything unknown
	// so that they can be re-parsed by the provider.
	parser := flags.NewParser(&opts, flags.IgnoreUnknown)

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
	var empty struct{}
	_, err = flags.ParseArgs(&empty, remain)
	if err != nil {
		log.Error(err)
		parser.WriteHelp(os.Stdout)
		os.Exit(1)
	}

	// Start the backup process
	err = Run(prov, &opts)
}

func Run(p provider.Provider, opts *config.GlobalOpts) (err error) {
	log := logrus.WithField("prefix", "rcon")

	log.Debug("creating client")
	client, err := rcon.CreateClient(opts)
	if err != nil {
		log.Warn("error creating client")
		return
	}

	// Disable automatic saving
	output, err := client.SendCommand("save-off")
	log.Info(output)
	if err == nil {

		// Manually save before taking backup
		output, err = client.SendCommand("save-all")
		log.Info(output)
		if err != nil {
			log.Error(err)
			log.Warn("saving failed, attempting to re-enable saving")
		} else {

			// Take a backup if saving succeeded
			err = p.TakeBackup()
			if err != nil {
				// Log the error but continue to re-enable saving.
				// Saving shouldn't ever be left disabled
				log.Warn(err)
			}
		}
	}

	// Always re-enable automatic saving
	output, err = client.SendCommand("save-on")
	if err != nil {
		return
	}
	log.Info(output)

	return
}
