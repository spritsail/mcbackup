package mcbackup

import (
	"github.com/sirupsen/logrus"
	"github.com/spritsail/mcbackup/config"
	"github.com/spritsail/mcbackup/provider"
	"github.com/spritsail/mcbackup/rcon"
)

type mcbackup struct {
	prov provider.Provider
	opts *config.GlobalOpts
}

func New(p provider.Provider, opts *config.GlobalOpts) *mcbackup {
	mb := new(mcbackup)
	mb.prov = p
	mb.opts = opts
	return mb
}

func (mb *mcbackup) Cron() {
	log := logrus.WithField("prefix", "cron")
	log.Info("starting cron")
}

func (mb *mcbackup) Run() (err error) {
	log := logrus.WithField("prefix", "backup")

	log.Debug("creating client")
	client, err := rcon.CreateClient(mb.opts)
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
			err = mb.prov.TakeBackup()
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
