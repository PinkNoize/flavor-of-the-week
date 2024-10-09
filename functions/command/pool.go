package command

import (
	"context"
	"fmt"

	"github.com/PinkNoize/flavor-of-the-week/functions/activity"
	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"github.com/PinkNoize/flavor-of-the-week/functions/customid"
	"github.com/PinkNoize/flavor-of-the-week/functions/utils"
	"github.com/bwmarrin/discordgo"
)

type PoolListCommand struct {
	GuildID      string
	Name         string
	ActivityType string
	CustomID     *customid.CustomID
}

func NewPoolListCommand(guildID, name, activityType string) *PoolListCommand {
	return &PoolListCommand{
		GuildID:      guildID,
		Name:         name,
		ActivityType: activityType,
	}
}

func NewPoolListCommandFromCustomID(guildID string, customID *customid.CustomID) *PoolListCommand {
	return &PoolListCommand{
		GuildID:      guildID,
		Name:         customID.Filter().Name,
		ActivityType: customID.Filter().Type,
		CustomID:     customID,
	}
}

func (c *PoolListCommand) Execute(ctx context.Context, cl *clients.Clients) (*discordgo.WebhookEdit, error) {
	actType := ""
	switch c.ActivityType {
	case "activity":
		actType = activity.ACTIVITY
	case "game":
		actType = activity.GAME
	default:
		actType = c.ActivityType
	}
	if c.CustomID == nil {
		customID, err := customid.CreateCustomID(ctx, "pool", customid.Filter{
			Name: c.Name,
			Type: actType,
		}, 0, cl)
		if err != nil {
			return nil, fmt.Errorf("CreateCustomID: %v", err)
		}
		c.CustomID = customID
	}

	entries, lastPage, err := activity.GetActivitiesPage(ctx, c.GuildID, c.CustomID.Page, &activity.ActivitesPageOptions{
		Name:            c.Name,
		Type:            activity.ActivityType(actType),
		NominationsOnly: false,
	}, cl)
	if err != nil {
		return nil, fmt.Errorf("GetActivitesPage: %v", err)
	}
	return utils.BuildDiscordPage(entries, c.CustomID, &utils.PageOptions{IsLastPage: lastPage}, nil), nil
}
