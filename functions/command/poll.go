package command

import (
	"context"
	"fmt"

	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"github.com/PinkNoize/flavor-of-the-week/functions/guild"
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
	g, err := guild.GetGuild(ctx, c.GuildID, cl)
	if err != nil {
		return nil, fmt.Errorf("GetGuild: %v", err)
	}
	err = g.SetPollChannel(ctx, c.ChannelID)
	if err != nil {
		return nil, fmt.Errorf("SetPollChannel: %v", err)
	}
	return utils.NewWebhookEdit(fmt.Sprintf("Set poll channel to <#%v>", c.ChannelID)), nil
}
