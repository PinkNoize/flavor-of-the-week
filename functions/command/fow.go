package command

import (
	"context"

	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"github.com/PinkNoize/flavor-of-the-week/functions/utils"
	"github.com/bwmarrin/discordgo"
)

type SetFowCommand struct {
	GuildID string
	Name    string
}

func NewSetFowCommand(guildID, name string) *SetFowCommand {
	return &SetFowCommand{
		GuildID: guildID,
		Name:    name,
	}
}

func (c *SetFowCommand) Execute(ctx context.Context, cl *clients.Clients) (*discordgo.WebhookEdit, error) {
	// TODO
	return utils.NewWebhookEdit("🚧 Command not implemented yet"), nil
}
