package provider

import (
	"strings"
	"time"

	"github.com/spritsail/mcbackup/backup"
	"github.com/spritsail/mcbackup/config"
)

type Provider interface {
	Create(name string, when time.Time) (backup.Backup, error)
	List() (backup.Backups, error)
}

var allProviders = map[string]func([]string, *config.Options) (Provider, []string, error){
	"zfs": NewZFS,
	"tar": NewTar,
}

func Register(name string, init func([]string, *config.Options) (Provider, []string, error)) {
	allProviders[name] = init
}

func Find(name string) func([]string, *config.Options) (Provider, []string, error) {
	return allProviders[strings.ToLower(name)]
}
