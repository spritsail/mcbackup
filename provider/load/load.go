package load

import (
	"github.com/spritsail/mcbackup/provider"
	"github.com/spritsail/mcbackup/provider/zfs"
	"strings"
)

var allProviders = map[string]func([]string) (provider.Provider, []string, error){
	"zfs": zfs.New,
}

func Find(name string) func([]string) (provider.Provider, []string, error) {
	return allProviders[strings.ToLower(name)]
}
