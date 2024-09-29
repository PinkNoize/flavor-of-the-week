package command

import (
	"context"
	"fmt"

	"github.com/PinkNoize/flavor-of-the-week/functions/activity"
	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"github.com/PinkNoize/flavor-of-the-week/functions/utils"
	"github.com/bwmarrin/discordgo"
)

type NominationAddCommand struct {
	GuildID string
	Name    string
	UserID  string
}

func NewNominationAddCommand(guildID, userID, name string) *NominationAddCommand {
	return &NominationAddCommand{
		GuildID: guildID,
		Name:    name,
		UserID:  userID,
	}
}

func (c *NominationAddCommand) Execute(ctx context.Context, cl *clients.Clients) (*discordgo.WebhookEdit, error) {
	act, err := activity.GetActivity(ctx, c.Name, c.GuildID, cl)
	if err != nil {
		ae, ok := err.(*activity.ActivityError)
		if ok && ae.Reason == activity.DOES_NOT_EXIST {
			return utils.NewWebhookEdit(fmt.Sprintf("%v does not exist", c.Name)), nil
		}
		return nil, err
	}
	err = act.AddNomination(ctx, c.UserID)
	if err != nil {
		return nil, fmt.Errorf("act.AddNomination: %v", err)
	}
	return utils.NewWebhookEdit(fmt.Sprintf("Added a nomination for %v", c.Name)), nil
}

type NominationRemoveCommand struct {
	GuildID string
	Name    string
	UserId  string
}

func NewNominationRemoveCommand(guildID, userID, name string) *NominationRemoveCommand {
	return &NominationRemoveCommand{
		GuildID: guildID,
		Name:    name,
		UserId:  userID,
	}
}

func (c *NominationRemoveCommand) Execute(ctx context.Context, cl *clients.Clients) (*discordgo.WebhookEdit, error) {
	act, err := activity.GetActivity(ctx, c.Name, c.GuildID, cl)
	if err != nil {
		ae, ok := err.(*activity.ActivityError)
		if ok && ae.Reason == activity.DOES_NOT_EXIST {
			return utils.NewWebhookEdit(fmt.Sprintf("%v does not exist", c.Name)), nil
		}
		return nil, err
	}
	err = act.RemoveNomination(ctx, c.UserId)
	if err != nil {
		return nil, fmt.Errorf("act.RemoveNomination: %v", err)
	}
	return utils.NewWebhookEdit(fmt.Sprintf("Removed a nomination for %v", c.Name)), nil
}

type NominationListCommand struct {
	GuildID string
	UserID  string
	Name    string
	Page    int
}

func NewNominationListCommand(guildID, userID, name string, page int) *NominationListCommand {
	return &NominationListCommand{
		GuildID: guildID,
		UserID:  userID,
		Name:    name,
		Page:    page,
	}
}

func (c *NominationListCommand) Execute(ctx context.Context, cl *clients.Clients) (*discordgo.WebhookEdit, error) {
	entries, lastPage, err := activity.GetActivitiesPage(ctx, c.GuildID, c.Page, &activity.ActivitesPageOptions{
		Name:            c.Name,
		NominationsOnly: true,
		UserId:          c.UserID,
	}, cl)
	if err != nil {
		return nil, fmt.Errorf("GetActivitesPage: %v", err)
	}
	customID := utils.NewCustomID("nominations", utils.Filter{
		Name: c.Name,
	}, c.Page)
	return utils.BuildDiscordPage(entries, customID, lastPage), nil
}
