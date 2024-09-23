package command

import (
	"context"

	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"github.com/bwmarrin/discordgo"
)

type NominationAddCommand struct {
	GuildID string
	Name    string
}

func NewNominationAddCommand(guildID, name string) *NominationAddCommand {
	return &NominationAddCommand{
		GuildID: guildID,
		Name:    name,
	}
}

func (c *NominationAddCommand) Execute(ctx context.Context, cl *clients.Clients) (*discordgo.WebhookParams, error) {
	// TODO
	return &discordgo.WebhookParams{
		Content: "🚧 Command not implemented yet",
		Flags:   discordgo.MessageFlagsEphemeral,
	}, nil
}

type NominationRemoveCommand struct {
	GuildID string
	Name    string
}

func NewNominationRemoveCommand(guildID, name string) *NominationRemoveCommand {
	return &NominationRemoveCommand{
		GuildID: guildID,
		Name:    name,
	}
}

func (c *NominationRemoveCommand) Execute(ctx context.Context, cl *clients.Clients) (*discordgo.WebhookParams, error) {
	// TODO
	return &discordgo.WebhookParams{
		Content: "🚧 Command not implemented yet",
		Flags:   discordgo.MessageFlagsEphemeral,
	}, nil
}

type NominationListCommand struct {
	GuildID string
	UserID  string
	Name    string
}

func NewNominationListCommand(guildID, userID, name string) *NominationListCommand {
	return &NominationListCommand{
		GuildID: guildID,
		UserID:  userID,
		Name:    name,
	}
}

func (c *NominationListCommand) Execute(ctx context.Context, cl *clients.Clients) (*discordgo.WebhookParams, error) {
	// TODO
	return &discordgo.WebhookParams{
		Content: "🚧 Command not implemented yet",
		Flags:   discordgo.MessageFlagsEphemeral,
	}, nil
}
