package command

import (
	"context"
	"fmt"

	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"github.com/PinkNoize/flavor-of-the-week/functions/guild"
	"github.com/PinkNoize/flavor-of-the-week/functions/utils"
	"github.com/bwmarrin/discordgo"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
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
	g, err := guild.GetGuild(ctx, c.GuildID, cl)
	if err != nil {
		return nil, fmt.Errorf("GetGuild: %v", err)
	}
	chanID, err := g.GetPollChannel(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetPollChannel: %v", err)
	}
	if chanID == nil {
		return utils.NewWebhookEdit("The poll channel has not been set"), nil
	}
	s, err := cl.Discord()
	if err != nil {
		return nil, fmt.Errorf("Discord: %v", err)
	}
	// REMOVE
	test, err := s.Application("@me")
	if err != nil {
		return nil, fmt.Errorf("Application: %v", err)
	}
	ctxzap.Info(ctx, test.Name)

	msg, err := s.ChannelMessageSendComplex(*chanID, &discordgo.MessageSend{
		Content: "hello",
	})
	if err != nil {
		return nil, fmt.Errorf("ChannelMessageSendComplex: %v", err)
	}
	// TODO: Remove nominations? Or only for winner?
	msgLink := fmt.Sprintf("https://discord.com/channels/%v/%v/%v", c.GuildID, chanID, msg.ID)
	return utils.NewWebhookEdit(fmt.Sprintf("Poll created: %v", msgLink)), nil
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
