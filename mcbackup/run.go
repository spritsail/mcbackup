package mcbackup

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/seeruk/minecraft-rcon/rcon"
	"github.com/sirupsen/logrus"
	"github.com/spritsail/mcbackup/config"
	"github.com/spritsail/mcbackup/mcbackup/cron"
	"github.com/spritsail/mcbackup/provider"
)

type mcbackup struct {
	prov provider.Provider
	rcon *rcon.Client
	opts *config.Options
}

func New(p provider.Provider, rc *rcon.Client, opts *config.Options) *mcbackup {
	mb := new(mcbackup)
	mb.prov = p
	mb.rcon = rc
	mb.opts = opts
	return mb
}

func (mb *mcbackup) Cron() {
	log := logrus.WithField("prefix", "cron")
	log.Info("starting cron")

	job, err := cron.Schedule(mb.opts.Cron.CronSchedule, mb.RunOnce)
	if err != nil {
		log.WithError(err).
			Fatal("failed to schedule backup job")
	}

	go job.Run()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigs:
		// Wait for a signal
		log.WithField("signal", sig).Info("caught signal")
		// Stop the repeated task and then wait for it to complete (below)
		log.Info("waiting for backup to complete")
		job.Cancel()

		// Now wait for job to terminate
		err = <-job.Done
		if err != nil {
			log.WithError(err).
				Warn("final backup run returned error")
		} else {
			log.Info("backup complete")
		}
		break

	case err = <-job.Done:
		// Wait for the job to complete
		break
	}
}

func (mb *mcbackup) RunOnce(when time.Time) (err error) {
	log := logrus.WithField("prefix", "backup")

	backupName, err := mb.opts.GenBackupName(when)
	if err != nil {
		return err
	}

	// Send a test command to check the client works
	_, err = mb.rcon.SendCommand("list")
	if err != nil {
		log.Error("error communicating with rcon, reconnecting")

		// Create a new client and try reconnecting
		mb.rcon, err = NewClient(mb.opts)

		// Only return error if reconnecting fails
		if err != nil {
			return
		}
	}

	log.Info("starting backup")

	// Disable automatic saving
	output, err := mb.rcon.SendCommand("save-off")
	log.Info(output)
	if err == nil {

		// Manually save before taking backup
		output, err = mb.rcon.SendCommand("save-all")
		log.Info(output)
		if err != nil {
			log.WithError(err).
				Warn("saving failed, attempting to re-enable saving")
		} else {
			if !mb.opts.DryRun {
				// Take a backup if saving succeeded
				err = mb.prov.Create(backupName)
			}
			if err != nil {
				// Log the error but continue to re-enable saving.
				// Saving shouldn't ever be left disabled
				log.WithError(err).
					Error("failed to take backup")
			}
		}
	}

	// Always re-enable automatic saving
	output, err = mb.rcon.SendCommand("save-on")
	if err != nil {
		return
	}
	log.Info(output)

	return
}

func NewClient(opts *config.Options) (*rcon.Client, error) {
	return rcon.NewClient(
		opts.Host,
		int(opts.Port),
		opts.Password,
	)
}
