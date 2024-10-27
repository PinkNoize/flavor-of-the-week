package functions

import (
	"context"
	"fmt"
	"time"

	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"github.com/PinkNoize/flavor-of-the-week/functions/command"
	"github.com/PinkNoize/flavor-of-the-week/functions/guild"
	"github.com/PinkNoize/flavor-of-the-week/functions/setup"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

func PollPubSub(ctx context.Context, _ event.Event) error {
	err := endActivePolls(ctx, setup.ClientLoader)
	if err != nil {
		setup.ZapSlogger.Errorf("endActivePolls: %v", err)
	}
	return nil
}

func endActivePolls(ctx context.Context, cl *clients.Clients) error {
	discordClient, err := cl.Discord()
	if err != nil {
		return fmt.Errorf("Discord: %v", err)
	}
	// Get all active polls
	guilds, err := guild.GetGuildsWithActivePolls(ctx, cl)
	if err != nil {
		return fmt.Errorf("GetGuildsWithActivePolls: %v", err)
	}
	prevContext := ctx
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
			// End it if active
			cmd := command.NewEndPollCommand(g.GetGuildId())
			_, err = cmd.Execute(ctx, cl)
			if err != nil {
				ctxzap.Warn(ctx, fmt.Sprintf("EndPollCommandExecute: %v", err))
				continue
			}
		}
	}
	return nil
}
