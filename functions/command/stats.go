package command

import (
	"context"
	"fmt"

	"github.com/PinkNoize/flavor-of-the-week/functions/activity"
	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"github.com/PinkNoize/flavor-of-the-week/functions/guild"
	"github.com/bwmarrin/discordgo"
)

type StatsCommand struct {
	GuildID string
}

func NewStatsCommand(guildID string) *StatsCommand {
	return &StatsCommand{
		GuildID: guildID,
	}
}

func (c *StatsCommand) Execute(ctx context.Context, cl *clients.Clients) (*discordgo.WebhookEdit, error) {
	// TODO: Display total pool count, # of all nominated items, FOW, # of FOWs
	g, err := guild.GetGuild(ctx, c.GuildID, cl)
	if err != nil {
		return nil, fmt.Errorf("GetGuild: %v", err)
	}
	discordSession, err := cl.Discord()
	if err != nil {
		return nil, fmt.Errorf("Discord: %v", err)
	}
	guildInfo, err := discordSession.Guild(c.GuildID)
	if err != nil {
		return nil, fmt.Errorf("Guild: %v", err)
	}
	fow, err := g.GetFow(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetFow: %v", err)
	}
	numFow, err := g.GetFowCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetFowCount: %v", err)
	}
	poolSize, err := activity.GetPoolSize(ctx, c.GuildID, cl)
	if err != nil {
		return nil, fmt.Errorf("GetPoolSize: %v", err)
	}
	return &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{
			{
				Title: guildInfo.Name,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:  "Flavor of the Week",
						Value: *fow,
					},
					{
						Name:   "# of FoWs",
						Value:  fmt.Sprint(numFow),
						Inline: true,
					},
					{
						Name:   "Pool size",
						Value:  fmt.Sprint(poolSize),
						Inline: true,
					},
				},
			},
		},
	}, nil
}
