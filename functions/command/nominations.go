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
	GuildID  string
	UserID   string
	Name     string
	CustomID *customid.CustomID
}

func NewNominationListCommand(guildID, userID, name string) *NominationListCommand {
	return &NominationListCommand{
		GuildID: guildID,
		UserID:  userID,
		Name:    name,
	}
}

func NewNominationListCommandFromCustomID(guildID, userID string, customID *customid.CustomID) *NominationListCommand {
	return &NominationListCommand{
		GuildID:  guildID,
		UserID:   userID,
		Name:     customID.Filter().Name,
		CustomID: customID,
	}
}

func (c *NominationListCommand) Execute(ctx context.Context, cl *clients.Clients) (*discordgo.WebhookEdit, error) {
	if c.CustomID == nil {
		typ := "nominations-list"
		if c.UserID != "" {
			typ = "nominations-mine"
		}
		customID, err := customid.CreateCustomID(ctx, typ, customid.Filter{
			Name: c.Name,
		}, 0, cl)
		if err != nil {
			return nil, fmt.Errorf("CreateCustomID: %v", err)
		}
		c.CustomID = customID
	}
	entries, lastPage, err := activity.GetActivitiesPage(ctx, c.GuildID, c.CustomID.Page, &activity.ActivitesPageOptions{
		Name:            c.Name,
		NominationsOnly: true,
		UserId:          c.UserID,
	}, cl)
	if err != nil {
		return nil, fmt.Errorf("GetActivitesPage: %v", err)
	}
	edit := utils.BuildDiscordPage(entries, c.CustomID, &utils.PageOptions{IsLastPage: lastPage}, nil)
	if edit.Embeds != nil && len(*edit.Embeds) == 0 {
		return &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{
				{
					Type:        discordgo.EmbedTypeImage,
					Description: "You got no nominations",
					Image: &discordgo.MessageEmbedImage{
						URL: "https://i.imgflip.com/95guyj.jpg",
					},
				},
			},
		}, nil
	}
	return edit, nil
}
