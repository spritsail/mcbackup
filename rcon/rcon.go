package rcon

import (
	"github.com/seeruk/minecraft-rcon/rcon"
	"github.com/spritsail/mcbackup/config"
)

func CreateClient(opts *config.GlobalOpts) (*rcon.Client, error) {
	client, err := rcon.NewClient(opts.Host, int(opts.Port), opts.Password)
	if err != nil {
		return nil, err
	}

	// Test the connection with a 'list' command
	// If no errors then we're good
	_, err = client.SendCommand("list")

	return client, err
}
