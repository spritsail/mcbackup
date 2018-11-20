package mcbackup

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SeerUK/minecraft-rcon/rcon"
	"github.com/dustin/go-humanize"
	"github.com/sirupsen/logrus"
	"github.com/spritsail/mcbackup/backup"
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

	job, err := cron.Schedule(mb.opts.Cron.CronSchedule, mb.cronRunner)
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
func (mb *mcbackup) cronRunner(t time.Time) error {
	err := mb.TakeBackup(t)
	if err != nil {
		return err
	}

	if !mb.opts.Cron.NoPrune {
		return mb.Prune(t)
	}
	return nil
}

func (mb *mcbackup) TakeBackup(when time.Time) (err error) {
	log := logrus.WithField("prefix", "rcon")

	backupName, err := mb.opts.GenBackupName(when)
	if err != nil {
		return err
	}

	// Send a test command to check the client works
	_, err = mb.rcon.SendCommand("list")
	if err != nil {
		log.Error("error communicating with rcon, reconnecting")

		// Try reconnecting
		err = mb.rcon.Reconnect()

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
				var bkup backup.Backup
				start := time.Now()
				bkup, err = mb.prov.Create(backupName, when)
				elapsed := time.Since(start)

				if err == nil {
					logBackupStats(bkup, elapsed)
				} else {
					// Log the error but don't return to re-enable saving.
					// Saving shouldn't ever be left disabled
					logrus.
						WithField("prefix", "backup").
						WithError(err).
						Error("failed to take backup")
				}
			}
		}
	}

	// Always re-enable automatic saving before returning
	output, e := mb.rcon.SendCommand("save-on")
	if e != nil {
		log.WithError(e).Warn(output)
		return e
	}
	log.Info(output)

	return
}

func logBackupStats(bkup backup.Backup, elapsed time.Duration) error {
	log := logrus.WithField("prefix", "backup")
	var hSize, hUsed = "?", "?"

	// Log size/disk space used
	size, err := bkup.Size()
	if err == nil {
		hSize = humanize.Bytes(size)
	} else {
		log.WithError(err).
			Warnf("failed to get size of backup")
		log.Infof("backup %s created in %s",
			bkup.Name(), elapsed)
		return err
	}

	used, err := bkup.SpaceUsed()
	if err == nil {
		hUsed = humanize.Bytes(used)
	} else {
		log.WithError(err).
			Warnf("failed to get space used by backup")
		log.Infof("backup %s created in %s, %s size",
			bkup.Name(), elapsed, hSize)
		return err
	}

	mbps := float64(used) / elapsed.Seconds()
	hMbps := humanize.Bytes(uint64(mbps))

	if size == used {
		log.Infof("backup %s created in %s (%s/s), %s size",
			bkup.Name(), elapsed, hMbps, hSize)
	} else {
		log.Infof("backup %s created in %s (%s/s), %s size, (%s on disk)",
			bkup.Name(), elapsed, hSize, hSize, hUsed)
	}

	return nil
}

func NewClient(opts *config.Options) (*rcon.Client, error) {
	return rcon.NewClient(
		opts.Host,
		int(opts.Port),
		opts.Password,
	)
}
