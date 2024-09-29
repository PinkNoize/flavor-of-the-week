package command

import (
	"context"
	"fmt"

	"github.com/PinkNoize/flavor-of-the-week/functions/activity"
	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"github.com/PinkNoize/flavor-of-the-week/functions/utils"
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

func (c *AddCommand) Execute(ctx context.Context, cl *clients.Clients) (*discordgo.WebhookEdit, error) {
	var typ activity.ActivityType
	switch c.ActivityType {
	case "activity":
		typ = activity.ACTIVITY
	case "game":
		return utils.NewWebhookEdit("🚧 Games not implemented yet"), nil
	default:
		return nil, fmt.Errorf("Activity type not supported: %v", c.ActivityType)
	}
	_, err := activity.Create(ctx, typ, c.Name, c.GuildID, cl)
	if err != nil {
		ae, ok := err.(*activity.ActivityError)
		if ok {
			if ae.Reason == activity.ALREADY_EXISTS {
				return utils.NewWebhookEdit(fmt.Sprintf("%v already exists in the pool", c.Name)), nil
			}
		}
		return nil, fmt.Errorf("act.Create: %v", err)
	}
	return utils.NewWebhookEdit(fmt.Sprintf("%v added to the pool", c.Name)), nil
}
