package command

import (
	"context"

	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"github.com/PinkNoize/flavor-of-the-week/functions/utils"
	"github.com/bwmarrin/discordgo"
)

type StatsCommand struct {
	GuildID string
}

func NewStatsCommand(guildID string) *StatsCommand {
	return &StatsCommand{
		GuildID: guildID,
	}
}

func (c *StatsCommand) Execute(ctx context.Context, cl *clients.Clients) (*discordgo.WebhookEdit, error) {
	// TODO: Display total pool count, # of all nominated items, FOW, # of FOWs
	return utils.NewWebhookEdit("🚧 Command not implemented yet"), nil
}
