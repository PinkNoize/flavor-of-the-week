package command

import (
	"context"
	"fmt"

	"github.com/PinkNoize/flavor-of-the-week/functions/activity"
	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"github.com/bwmarrin/discordgo"
)

type AddCommand struct {
	GuildID      string
	ActivityType string
	Name         string
}

func NewAddCommand(guildID, activityType, name string) *AddCommand {
	return &AddCommand{
		GuildID:      guildID,
		ActivityType: activityType,
		Name:         name,
	}
}

func (c *AddCommand) Execute(ctx context.Context, cl *clients.Clients) (*discordgo.InteractionResponse, error) {
	var typ activity.ActivityType
	switch c.ActivityType {
	case "activity":
		typ = activity.ACTIVITY
	// TODO: check activity type, if game lookup, validate first
	default:
		return nil, fmt.Errorf("Activity type not supported: %v", c.ActivityType)
	}
	_, err := activity.Create(ctx, typ, c.Name, c.GuildID, cl)
	if err != nil {
		ae, ok := err.(*activity.ActivityError)
		if ok {
			if ae.Reason == activity.ALREADY_EXISTS {
				return &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("%v already exists in the pool", c.Name),
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				}, nil
			}
		}
		return nil, fmt.Errorf("act.Create: %v", err)
	}
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("%v added to the pool", c.Name),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}, nil
}
