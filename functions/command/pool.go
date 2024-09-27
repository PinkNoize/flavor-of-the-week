package command

import (
	"context"
	"fmt"

	"github.com/PinkNoize/flavor-of-the-week/functions/activity"
	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"github.com/PinkNoize/flavor-of-the-week/functions/utils"
	"github.com/bwmarrin/discordgo"
)

type PoolListCommand struct {
	GuildID      string
	Name         string
	ActivityType string
	Page         int
}

func NewPoolListCommand(guildID, name, activityType string, page int) *PoolListCommand {
	return &PoolListCommand{
		GuildID:      guildID,
		Name:         name,
		ActivityType: activityType,
		Page:         page,
	}
}

func (c *PoolListCommand) Execute(ctx context.Context, cl *clients.Clients) (*discordgo.WebhookEdit, error) {
	entries, lastPage, err := activity.GetActivitiesPage(ctx, c.GuildID, c.Page, &activity.ActivitesPageOptions{
		Name:            c.Name,
		Type:            activity.ActivityType(c.ActivityType),
		NominationsOnly: false,
	}, cl)
	if err != nil {
		return nil, fmt.Errorf("GetActivitesPage: %v", err)
	}
	return utils.BuildDiscordPage(entries, "pool", c.Page, lastPage), nil
}
