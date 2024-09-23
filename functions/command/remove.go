package command

import (
	"context"

	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"github.com/bwmarrin/discordgo"
)

type RemoveCommand struct {
	GuildID string
	Name    string
}

func NewRemoveCommand(guildID, name string) *RemoveCommand {
	return &RemoveCommand{
		GuildID: guildID,
		Name:    name,
	}
}

func (c *RemoveCommand) Execute(ctx context.Context, cl *clients.Clients) (*discordgo.WebhookParams, error) {
	// TODO
	return &discordgo.WebhookParams{
		Content: "🚧 Command not implemented yet",
		Flags:   discordgo.MessageFlagsEphemeral,
	}, nil
}
