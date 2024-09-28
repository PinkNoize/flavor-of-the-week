package command

import (
	"context"
	"fmt"

	"github.com/PinkNoize/flavor-of-the-week/functions/activity"
	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"github.com/PinkNoize/flavor-of-the-week/functions/utils"
	"github.com/bwmarrin/discordgo"
)

type RemoveCommand struct {
	GuildID string
	Name    string
	Force   bool
}

func NewRemoveCommand(guildID, name string, force bool) *RemoveCommand {
	return &RemoveCommand{
		GuildID: guildID,
		Name:    name,
		Force:   force,
	}
}

func (c *RemoveCommand) Execute(ctx context.Context, cl *clients.Clients) (*discordgo.WebhookEdit, error) {
	act, err := activity.GetActivity(ctx, c.Name, c.GuildID, cl)
	if err != nil {
		ae, ok := err.(*activity.ActivityError)
		if ok && ae.Reason == activity.DOES_NOT_EXIST {
			return utils.NewWebhookEdit(fmt.Sprintf("%v does not exist", c.Name)), nil
		}
		return nil, err
	}
	err = act.RemoveActivity(ctx, c.Force)
	if err != nil {
		ae, ok := err.(*activity.ActivityError)
		if ok && ae.Reason == activity.STILL_HAS_NOMINATIONS {
			return utils.NewWebhookEdit(fmt.Sprintf("Cannot remove %v as it has nominations", c.Name)), nil
		}
		return nil, fmt.Errorf("act.RemoveActivity: %v", err)
	}
	return utils.NewWebhookEdit(fmt.Sprintf("Removed %v from the pool", c.Name)), nil
}
