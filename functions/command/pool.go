package command

import (
	"context"

	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"github.com/bwmarrin/discordgo"
)

type PoolListCommand struct {
	GuildID      string
	Name         string
	ActivityType string
}

func NewPoolListCommand(guildID, name, activityType string) *PoolListCommand {
	return &PoolListCommand{
		GuildID:      guildID,
		Name:         name,
		ActivityType: activityType,
	}
}

func (c *PoolListCommand) Execute(ctx context.Context, cl *clients.Clients) (*discordgo.WebhookParams, error) {
	// TODO
	return &discordgo.WebhookParams{
		Content: "🚧 Command not implemented yet",
		Flags:   discordgo.MessageFlagsEphemeral,
	}, nil
}
