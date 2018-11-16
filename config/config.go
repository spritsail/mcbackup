package config

import (
	"time"

	"github.com/knz/strtime"
)

type GlobalOpts struct {
	Host         string `short:"h" long:"host" description:"Minecraft server host address" env:"RCON_HOST" required:"true"`
	Port         uint   `short:"p" long:"port" description:"Minecraft server RCON port" env:"RCON_PORT" default:"25575"`
	Password     string `short:"P" long:"password" description:"Minecraft server RCON password" env:"RCON_PASS" required:"true"`
	Provider     string `long:"provider" description:"Backup provider, for taking/storing backups" env:"BACKUP_PROVIDER" default:"tar" choice:"zfs" choice:"zip" choice:"tar"`
	BackupPrefix string `long:"backup-prefix" description:"Identifying prefix for mcbackup-managed backups" env:"BACKUP_PREFIX" default:"mcb-"`
	BackupFormat string `long:"date-format" description:"Format for snapshot names (see date(1))" env:"BACKUP_FORMAT" default:"%F-%H:%M"`
	LogLevel     string `short:"l" long:"level" description:"log level verbosity" env:"LOG_LEVEL" choice:"warn" choice:"info" choice:"debug" choice:"trace" default:"info"`
	Cron         struct {
		CronSchedule string `command:"cron" short:"s" long:"cron-schedule" description:"Cron-like schedule to run backups on" env:"CRON_SCHEDULE" default:"*/15 * * * *"`
	} `command:"cron"`
	Run struct {
	} `command:"once"`
}

func (opts GlobalOpts) GenBackupName(when time.Time) (name string, err error) {
	// Generate backup name from prefix and date format
	formatted, err := strtime.Strftime(when, opts.BackupFormat)
	if err != nil {
		return
	}
	name = opts.BackupPrefix + formatted
	return
}
func (opts GlobalOpts) ParseBackupName(name string) (when time.Time, err error) {
	// Parse backup name from string and date format
	return strtime.Strptime(name, opts.BackupPrefix+opts.BackupFormat)
}
