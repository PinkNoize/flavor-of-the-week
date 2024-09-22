package command

import (
	"context"

	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
)

type RemoveCommand struct {
	guildID string
	name    string
}

func NewRemoveCommand(guildID, name string) *RemoveCommand {
	return &RemoveCommand{
		guildID: guildID,
		name:    name,
	}
}

func (c *RemoveCommand) Execute(ctx context.Context, cl *clients.Clients) error {
	// TODO
	return nil
}
