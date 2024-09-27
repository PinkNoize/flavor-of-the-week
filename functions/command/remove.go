package command

import (
	"context"
	"fmt"

	"github.com/PinkNoize/flavor-of-the-week/functions/activity"
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

func (c *RemoveCommand) Execute(ctx context.Context, cl *clients.Clients) (*discordgo.InteractionResponse, error) {
	act, err := activity.GetActivity(ctx, c.Name, c.GuildID, cl)
	if err != nil {
		ae, ok := err.(*activity.ActivityError)
		if ok && ae.Reason == activity.DOES_NOT_EXIST {
			return &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("%v does not exist", c.Name),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			}, nil
		}
		return nil, err
	}
	err = act.RemoveActivity(ctx)
	if err != nil {
		ae, ok := err.(*activity.ActivityError)
		if ok && ae.Reason == activity.STILL_HAS_NOMINATIONS {
			return &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("Cannot remove %v as it has nominations", c.Name),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			}, nil
		}
		return nil, fmt.Errorf("act.RemoveActivity: %v", err)
	}
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Removed %v from the pool", c.Name),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}, nil
}
