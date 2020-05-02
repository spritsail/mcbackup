package prometheus

import (
	"fmt"
	"sort"
	"time"

	"github.com/gorhill/cronexpr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spritsail/mcbackup/config"
	"github.com/spritsail/mcbackup/provider"
)

type BackupCollector struct {
	Provider provider.Provider
	Interval time.Duration

	backupLatest   *prometheus.Desc
	backupOldest   *prometheus.Desc
	backupCount    *prometheus.Desc
	backupInterval *prometheus.Desc
}

func newBackupCollector(prov provider.Provider, opts config.Options) BackupCollector {
	labels := prometheus.Labels{
		"mcserver": fmt.Sprintf("%s:%d", opts.Host, opts.Port),
		"provider": opts.Provider,
	}

	// Hacky nonsense to work out a rough interval for a cron expression
	// It would definitely be easier to use a fixed interval to begin with 🤷
	var interval time.Duration = 0
	if opts.Cron.CronSchedule != "" {
		next := cronexpr.MustParse(opts.Cron.CronSchedule).NextN(time.Now(), 2)
		interval = next[1].Sub(next[0])
	}

	return BackupCollector{
		Provider: prov,
		Interval: interval,

		backupLatest: prometheus.NewDesc("mcbackup_backup_latest",
			"Unix timestamp of the last successful backup",
			nil, labels,
		),
		backupOldest: prometheus.NewDesc("mcbackup_backup_oldest",
			"Unix timestamp of the oldest successful backup",
			nil, labels,
		),
		backupCount: prometheus.NewDesc("mcbackup_backup_count",
			"Number of backups stored in total",
			nil, labels,
		),
		backupInterval: prometheus.NewDesc("mcbackup_backup_interval",
			"Interval between backups",
			nil, labels,
		),
	}
}

func (b BackupCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- b.backupLatest
	ch <- b.backupOldest
	ch <- b.backupCount
	if b.Interval != 0 {
		ch <- b.backupInterval
	}
}

func (b BackupCollector) Collect(ch chan<- prometheus.Metric) {
	backups, err := b.Provider.List()
	if err != nil {
		log.WithError(err).Warn("error collecting latest metrics")
		return
	}

	// Sort so we can find the newest and oldest latest
	sort.Sort(backups)

	latest := backups[len(backups)-1]
	oldest := backups[0]

	ch <- prometheus.MustNewConstMetric(b.backupLatest, prometheus.CounterValue, float64(latest.When().Unix()))
	ch <- prometheus.MustNewConstMetric(b.backupOldest, prometheus.CounterValue, float64(oldest.When().Unix()))
	ch <- prometheus.MustNewConstMetric(b.backupCount, prometheus.GaugeValue, float64(len(backups)))
	if b.Interval != 0 {
		ch <- prometheus.MustNewConstMetric(b.backupInterval, prometheus.GaugeValue, b.Interval.Seconds())
	}
}

var _ prometheus.Collector = BackupCollector{}
