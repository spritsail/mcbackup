package config

type GlobalOpts struct {
	Host     string `short:"h" long:"host" description:"Minecraft server host address" env:"MCBACKUP_HOST" required:"true"`
	Port     uint   `short:"p" long:"port" description:"Minecraft server RCON port" env:"MCBACKUP_PORT" default:"25575"`
	Password string `short:"P" long:"password" description:"Minecraft server RCON password" env:"MCBACKUP_PASS" required:"true"`
	Provider string `long:"provider" description:"Backup provider, for taking/storing backups" env:"MCBACKUP_PROVIDER" default:"tar" choice:"zfs" choice:"zip" choice:"tar"`
	Cron     struct {
		CronSchedule string `command:"cron" short:"s" long:"cron-schedule" description:"Cron-like schedule to run backups on" env:"CRON_SCHEDULE" default:"*/15 * * * *"`
	} `command:"cron"`
	Run struct {
	} `command:"once"`
}
