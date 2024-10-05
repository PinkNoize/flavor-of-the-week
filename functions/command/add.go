package command

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/PinkNoize/flavor-of-the-week/functions/activity"
	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"github.com/PinkNoize/flavor-of-the-week/functions/utils"
	"github.com/bwmarrin/discordgo"
	"github.com/dimuska139/rawg-sdk-go/v3"
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
	var info *activity.GameInfo
	switch c.ActivityType {
	case "activity":
		typ = activity.ACTIVITY
	case "game":
		typ = activity.GAME

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Millisecond*500))
		defer cancel()

		detail, err := cl.Rawg().GetGame(ctx, c.Name)

		if err != nil {
			if rawgError, ok := err.(*rawg.RawgError); ok && rawgError.HttpCode == http.StatusNotFound {
				return utils.NewWebhookEdit("🚧 Game not found 🚧"), err
			} else {
				return utils.NewWebhookEdit("🤖🔥 Bot error 🔥"), err
			}
		}
		if detail == nil {
			return utils.NewWebhookEdit("🚧 Game not found 🚧"), err
		}

		info = &activity.GameInfo{
			Id:              strconv.Itoa(detail.ID),
			Slug:            detail.Slug,
			BackgroundImage: detail.ImageBackground,
		}
	default:
		return nil, fmt.Errorf("Activity type not supported: %v", c.ActivityType)
	}
	_, err := activity.Create(ctx, typ, c.Name, c.GuildID, info, cl)
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
