package provider

import (
	"strings"
)

type Provider interface {
	Create(name string) error
}

var allProviders = map[string]func([]string) (Provider, []string, error){
	"zfs": NewZFS,
}

func Find(name string) func([]string) (Provider, []string, error) {
	return allProviders[strings.ToLower(name)]
}
