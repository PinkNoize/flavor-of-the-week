package command

import (
	"context"

	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"github.com/PinkNoize/flavor-of-the-week/functions/utils"
	"github.com/bwmarrin/discordgo"
)

type StartPollCommand struct {
	GuildID string
}

func NewStartPollCommand(guildID string) *StartPollCommand {
	return &StartPollCommand{
		GuildID: guildID,
	}
}

func (c *StartPollCommand) Execute(ctx context.Context, cl *clients.Clients) (*discordgo.WebhookEdit, error) {
	// TODO
	return utils.NewWebhookEdit("🚧 Command not implemented yet"), nil
}

type SetPollChannelCommand struct {
	GuildID   string
	ChannelID string
}

func NewSetPollChannelCommand(guildID string, channel *discordgo.Channel) *SetPollChannelCommand {
	return &SetPollChannelCommand{
		GuildID:   guildID,
		ChannelID: channel.ID,
	}
}

func (c *SetPollChannelCommand) Execute(ctx context.Context, cl *clients.Clients) (*discordgo.WebhookEdit, error) {
	// TODO
	return utils.NewWebhookEdit("🚧 Command not implemented yet"), nil
}
