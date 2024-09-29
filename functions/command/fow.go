package command

import (
	"context"
	"fmt"

	"github.com/PinkNoize/flavor-of-the-week/functions/activity"
	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"github.com/PinkNoize/flavor-of-the-week/functions/guild"
	"github.com/PinkNoize/flavor-of-the-week/functions/utils"
	"github.com/bwmarrin/discordgo"
)

type SetFowCommand struct {
	GuildID string
	Name    string
}

func NewSetFowCommand(guildID, name string) *SetFowCommand {
	return &SetFowCommand{
		GuildID: guildID,
		Name:    name,
	}
}

func (c *SetFowCommand) Execute(ctx context.Context, cl *clients.Clients) (*discordgo.WebhookEdit, error) {
	g, err := guild.GetGuild(ctx, c.GuildID, cl)
	if err != nil {
		return nil, fmt.Errorf("GetGuild: %v", err)
	}
	// Check fow exists
	_, err = activity.GetActivity(ctx, c.Name, c.GuildID, cl)
	if err != nil {
		ae, ok := err.(*activity.ActivityError)
		if ok && ae.Reason == activity.DOES_NOT_EXIST {
			return utils.NewWebhookEdit(fmt.Sprintf("%v does not exist", c.Name)), nil
		}
		return nil, err
	}

	err = g.SetFow(ctx, c.Name)
	if err != nil {
		return nil, fmt.Errorf("SetFow: %v", err)
	}
	return utils.NewWebhookEdit(fmt.Sprintf("Set flavor of the week to %v", c.Name)), nil

}
