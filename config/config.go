package config

import (
	"strings"
	"time"

	"github.com/knz/strtime"
)

type Options struct {
	Host         string `short:"H" long:"host" description:"Minecraft server host address" env:"RCON_HOST" required:"true"`
	Port         uint   `short:"p" long:"port" description:"Minecraft server RCON port" env:"RCON_PORT" default:"25575"`
	Password     string `short:"P" long:"password" description:"Minecraft server RCON password" env:"RCON_PASS" required:"true"`
	Provider     string `long:"provider" description:"Backup provider, for taking/storing backups" env:"BACKUP_PROVIDER" default:"tar" choice:"zfs" choice:"zip" choice:"tar"`
	DryRun       bool   `short:"d" long:"dry-run" description:"Prevent performing any potentially catastrophic operations, only simulate them"`
	BackupPrefix string `long:"backup-prefix" description:"Identifying prefix for mcbackup-managed backups" env:"BACKUP_PREFIX" default:"mcb-"`
	BackupFormat string `long:"date-format" description:"Format for snapshot names (see date(1))" env:"BACKUP_FORMAT" default:"%F-%H:%M"`
	LogLevel     string `short:"l" long:"level" description:"log level verbosity" env:"LOG_LEVEL" choice:"warn" choice:"info" choice:"debug" choice:"trace" default:"info"`

	Cron struct {
		Prune
		CronSchedule string `short:"s" long:"cron-schedule" description:"Cron-like schedule to run backups on" env:"CRON_SCHEDULE" default:"*/15 * * * *"`
		NoPrune      bool   `long:"no-prune" description:"disable pruning during cron operation" env:"CRON_NO_PRUNE"`
	} `command:"cron"`

	Run struct {
	} `command:"once"`

	Prune Prune `command:"prune"`
}

// Prune tracks how many backup should be kept of each age
// By default it will keep the n most recent from each category
type Prune struct {
	KeepFor     time.Duration `short:"k" long:"keep" description:"length of time to keep all backups" env:"KEEP" default:"24h"`
	KeepHourly  uint          `long:"keep-hourly" description:"number of hours to keep a backup for" env:"KEEP_HOURLY" default:"12"`
	KeepDaily   uint          `long:"keep-daily" description:"number of days to keep a backup for" env:"KEEP_DAILY" default:"7"`
	KeepWeekly  uint          `long:"keep-weekly" description:"number of weeks to keep a backup for" env:"KEEP_WEEKLY" default:"4"`
	KeepMonthly uint          `long:"keep-monthly" description:"number of months to keep a backup for" env:"KEEP_MONTHLY" default:"6"`
	KeepYearly  uint          `long:"keep-yearly" description:"number of years to keep a backup for" env:"KEEP_YEARLY" default:"5"`
}

func (opts Options) GenBackupName(when time.Time) (name string, err error) {
	// Generate backup name from prefix and date format
	formatted, err := strtime.Strftime(when, opts.BackupFormat)
	if err != nil {
		return
	}
	name = opts.BackupPrefix + formatted
	return
}
func (opts Options) ParseBackupName(name string) (when time.Time, err error) {
	// Parse backup name from string and date format
	return strtime.Strptime(name, opts.BackupPrefix+opts.BackupFormat)
}
func (opts Options) IsMcbackup(name string) bool {
	return strings.HasPrefix(name, opts.BackupPrefix)
}
