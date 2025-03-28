package functions

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"github.com/PinkNoize/flavor-of-the-week/functions/command"
	"github.com/PinkNoize/flavor-of-the-week/functions/guild"
	"github.com/PinkNoize/flavor-of-the-week/functions/setup"
	"github.com/bwmarrin/discordgo"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

func PollPubSub(ctx context.Context, _ PubSubMessage) error {
	var err error
	logger, slogger := setup.ZapLogger, setup.ZapSlogger
	defer func() {
		err = errors.Join(slogger.Sync())
		err = errors.Join(logger.Sync())
	}()
	ctx = ctxzap.ToContext(ctx, logger)

	now := time.Now().UTC()

	ctxzap.Info(ctx, "Starting poll job")
	err = startScheduledPolls(ctx, now, setup.ClientLoader)
	if err != nil {
		slogger.Errorf("startScheduledPolls: %v", err)
	}
	err = notifyUpcomingPolls(ctx, now, setup.ClientLoader)
	if err != nil {
		slogger.Errorf("notifyUpcomingPolls: %v", err)
	}
	err = endActivePolls(ctx, setup.ClientLoader)
	if err != nil {
		slogger.Errorf("endActivePolls: %v", err)
	}
	return nil
}

func endActivePolls(ctx context.Context, cl *clients.Clients) error {
	discordClient, err := cl.Discord()
	if err != nil {
		return fmt.Errorf("discord: %v", err)
	}
	// Get all active polls
	guilds, err := guild.GetGuildsWithActivePolls(ctx, cl)
	if err != nil {
		return fmt.Errorf("GetGuildsWithActivePolls: %v", err)
	}
	ctxzap.Info(ctx, fmt.Sprintf("Found %v active polls", len(guilds)))
	prevContext := ctx
	if len(guilds) > 0 {
		time.Sleep(time.Minute)
	}

	for _, g := range guilds {
		ctx = prevContext
		ctxzap.AddFields(ctx, zap.String("guildID", g.GetGuildId()))
		activePoll, err := g.GetActivePoll(ctx)
		if err != nil {
			return fmt.Errorf("GetActivePoll: %v", err)
		}
		// Check if its active
		msg, err := discordClient.ChannelMessage(activePoll.ChannelID, activePoll.MessageID)
		if err != nil {
			ctxzap.Warn(ctx, fmt.Sprintf("Failed to get channel message for %v: %v %v", g.GetGuildId(), activePoll.ChannelID, activePoll.MessageID))
			continue
		}
		if msg.Poll == nil {
			ctxzap.Warn(ctx, fmt.Sprintf("No poll for guild %v", g.GetGuildId()))
			continue
		}
		if msg.Poll.Expiry != nil && msg.Poll.Expiry.Before(time.Now()) {
			ctxzap.Info(ctx, fmt.Sprintf("Poll for %v has ended. Ending poll", g.GetGuildId()))
			// End it if active
			cmd := command.NewEndPollCommand(g.GetGuildId())
			_, err = cmd.Execute(ctx, cl)
			if err != nil {
				ctxzap.Warn(ctx, fmt.Sprintf("EndPollCommandExecute: %v", err))
				continue
			}
		} else {
			ctxzap.Info(ctx, fmt.Sprintf("Poll for %v has not ended", g.GetGuildId()))
		}
	}
	return nil
}

func startScheduledPolls(ctx context.Context, now time.Time, cl *clients.Clients) error {
	day := now.Weekday()
	hour := now.Hour()
	ctxzap.Info(ctx, fmt.Sprintf("Searching for schedules with Day %v, Hour %v", day, hour))
	guilds, err := guild.GetGuildsWithSchedule(ctx, day, hour, cl)
	if err != nil {
		return fmt.Errorf("GetGuildsWithSchedule: %v", err)
	}

	ctxzap.Info(ctx, fmt.Sprintf("Found %v scheduled polls", len(guilds)))
	prevContext := ctx
	for _, g := range guilds {
		ctx = prevContext
		ctxzap.AddFields(ctx, zap.String("guildID", g.GetGuildId()))

		cmd := command.NewStartPollCommand(g.GetGuildId())
		_, err = cmd.Execute(ctx, cl)
		if err != nil {
			ctxzap.Warn(ctx, fmt.Sprintf("StartPollCommand: %v", err))
			continue
		}
	}
	return nil
}

func notifyUpcomingPolls(ctx context.Context, now time.Time, cl *clients.Clients) error {
	discordSession, err := cl.Discord()
	if err != nil {
		return fmt.Errorf("discord: %v", err)
	}

	tomorrow := now.Add(24 * time.Hour)
	day := tomorrow.Weekday()
	hour := tomorrow.Hour()
	ctxzap.Info(ctx, fmt.Sprintf("Searching for schedules with Day %v, Hour %v for notif", day, hour))
	guilds, err := guild.GetGuildsWithSchedule(ctx, day, hour, cl)
	if err != nil {
		return fmt.Errorf("GetGuildsWithSchedule: %v", err)
	}

	ctxzap.Info(ctx, fmt.Sprintf("Found %v scheduled polls", len(guilds)))
	prevContext := ctx
	for _, g := range guilds {
		ctx = prevContext
		ctxzap.AddFields(ctx, zap.String("guildID", g.GetGuildId()))

		pollChannel, err := g.GetPollChannel(ctx)
		if err != nil {
			ctxzap.Warn(ctx, fmt.Sprintf("GetPollChannel: %v", err))
			continue
		}
		_, err = discordSession.ChannelMessageSendComplex(*pollChannel, &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "⏳24 hours until the next poll⌛",
					Description: "Get your nominations in before it's too late.\nType */nominations* to get started.",
					Color:       2326507,
				},
			},
		})
		if err != nil {
			ctxzap.Warn(ctx, fmt.Sprintf("ChannelMessageSendComplex: %v", err))
			continue
		}
	}
	return nil
}
