package provider

import (
	"strings"

	"github.com/spritsail/mcbackup/backup"
	"github.com/spritsail/mcbackup/config"
)

type Provider interface {
	Create(name string) error
	List() (backup.Backups, error)
	Remove(name backup.Backup) error
}

var allProviders = map[string]func([]string, *config.Options) (Provider, []string, error){
	"zfs": NewZFS,
}

func Register(name string, init func([]string, *config.Options) (Provider, []string, error)) {
	allProviders[name] = init
}

func Find(name string) func([]string, *config.Options) (Provider, []string, error) {
	return allProviders[strings.ToLower(name)]
}
