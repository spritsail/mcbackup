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

/*
func Init(ctx *cli.Context) (provider.Provider, error) {
	name := strings.ToLower(ctx.String("provider"))

	providerInit := allProviders[name]
	if providerInit == nil {
		return nil, errors.New(fmt.Sprintf("No such provider '%s'", name))
	}

	return providerInit(ctx)
}
*/
