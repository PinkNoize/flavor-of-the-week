package command

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/PinkNoize/flavor-of-the-week/functions/activity"
	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"github.com/PinkNoize/flavor-of-the-week/functions/guild"
	"github.com/PinkNoize/flavor-of-the-week/functions/utils"
	"github.com/bwmarrin/discordgo"
	"github.com/cenkalti/backoff/v4"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

type StartPollCommand struct {
	GuildID string
}

func NewStartPollCommand(guildID string) *StartPollCommand {
	return &StartPollCommand{
		GuildID: guildID,
	}
}

func (c *StartPollCommand) Execute(ctx context.Context, cl *clients.Clients) (*discordgo.WebhookEdit, error) {
	g, err := guild.GetGuild(ctx, c.GuildID, cl)
	if err != nil {
		return nil, fmt.Errorf("GetGuild: %v", err)
	}
	chanID, err := g.GetPollChannel(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetPollChannel: %v", err)
	}
	if chanID == nil {
		return utils.NewWebhookEdit("The poll channel has not been set"), nil
	}
	pollID, err := g.GetActivePoll(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetActivePollID: %v", err)
	}
	if pollID != nil {
		return utils.NewWebhookEdit("There is already an active poll"), nil
	}

	s, err := cl.Discord()
	if err != nil {
		return nil, fmt.Errorf("Discord: %v", err)
	}

	msg, err := s.ChannelMessageSendComplex(*chanID, &discordgo.MessageSend{
		Poll: &discordgo.Poll{
			Question: discordgo.PollMedia{
				Text: "What should the flavor of the week be?",
			},
			Answers: []discordgo.PollAnswer{
				{
					Media: &discordgo.PollMedia{
						Text: "First",
					},
				},
				{
					Media: &discordgo.PollMedia{
						Text: "Two",
					},
				},
			},
			AllowMultiselect: false,
			LayoutType:       discordgo.PollLayoutTypeDefault,
			Duration:         1,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("ChannelMessageSendComplex: %v", err)
	}
	err = g.SetActivePoll(ctx, &guild.PollInfo{
		ChannelID: *chanID,
		MessageID: msg.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("SetActivePoll: %v", err)
	}
	msgLink := fmt.Sprintf("https://discord.com/channels/%v/%v/%v", c.GuildID, *chanID, msg.ID)
	return utils.NewWebhookEdit(fmt.Sprintf("Poll created: %v", msgLink)), nil
}

type EndPollCommand struct {
	GuildID string
}

func NewEndPollCommand(guildID string) *EndPollCommand {
	return &EndPollCommand{
		GuildID: guildID,
	}
}

func (c *EndPollCommand) Execute(ctx context.Context, cl *clients.Clients) (*discordgo.WebhookEdit, error) {
	g, err := guild.GetGuild(ctx, c.GuildID, cl)
	if err != nil {
		return nil, fmt.Errorf("GetGuild: %v", err)
	}
	pollID, err := g.GetActivePoll(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetActivePollID: %v", err)
	}
	if pollID == nil {
		return utils.NewWebhookEdit("There is no active poll to end"), nil
	}
	s, err := cl.Discord()
	if err != nil {
		return nil, fmt.Errorf("Discord: %v", err)
	}
	msg, err := s.ChannelMessage(pollID.ChannelID, pollID.MessageID)
	if err != nil || msg.Poll == nil {
		return utils.NewWebhookEdit("⚠️ Unable to retrieve the poll"), fmt.Errorf("ChannelMessage: %v", err)
	}
	ctxzap.Info(ctx, "Poll results", zap.Any("poll", *msg.Poll))
	if msg.Poll.Results == nil || !msg.Poll.Results.Finalized {
		msg, err = s.PollExpire(pollID.ChannelID, pollID.MessageID)
		if err != nil {
			return utils.NewWebhookEdit("⚠️ Unable to end the poll"), fmt.Errorf("PollExpire: %v", err)
		}
		waitForResults := func() error {
			msg, err = s.ChannelMessage(pollID.ChannelID, pollID.MessageID)
			if err != nil || msg.Poll == nil {
				return fmt.Errorf("ChannelMessage: %v", err)
			}
			ctxzap.Info(ctx, "Poll results", zap.Any("poll", *msg.Poll))
			if msg.Poll.Results == nil || !msg.Poll.Results.Finalized || msg.Poll.Results.AnswerCounts == nil {
				return fmt.Errorf("Poll not finalized")
			}
			return nil
		}
		err = backoff.Retry(waitForResults, backoff.NewExponentialBackOff(backoff.WithInitialInterval(time.Millisecond*750), backoff.WithMaxElapsedTime(time.Second*30)))
		if err != nil {
			return utils.NewWebhookEdit("Failed to end the poll"), fmt.Errorf("waitForResults: %v", err)
		}
		if msg.Poll.Results == nil || !msg.Poll.Results.Finalized || msg.Poll.Results.AnswerCounts == nil {
			return utils.NewWebhookEdit("Failed to get the poll results"), nil
		}
	}
	ctxzap.Info(ctx, "Poll results", zap.Any("poll", *msg.Poll))
	winner, tie := determinePollWinner(msg.Poll)
	var response *discordgo.WebhookEdit
	if tie {
		response = utils.NewWebhookEdit("Ended the poll with a tie")
	} else {
		err = g.SetFow(ctx, winner)
		if err != nil {
			return nil, fmt.Errorf("SetFow: %v", err)
		}
		response = utils.NewWebhookEdit(fmt.Sprintf("Poll ended\nWinner: %v", winner))
	}
	err = g.ClearActivePoll(ctx)
	if err != nil {
		return nil, fmt.Errorf("ClearActivePoll: %v", err)
	}
	err = activity.ClearNominations(ctx, c.GuildID, cl)
	if err != nil {
		return nil, fmt.Errorf("ClearNominations: %v", err)
	}
	return response, nil
}

func determinePollWinner(poll *discordgo.Poll) (string, bool) {
	answerCounts := poll.Results.AnswerCounts
	sort.Slice(answerCounts, func(i, j int) bool {
		if answerCounts[i] == nil && answerCounts[j] != nil {
			return true
		} else if answerCounts[i] != nil && answerCounts[j] == nil {
			return false
		} else if answerCounts[i] == nil && answerCounts[j] == nil {
			return false
		}
		return answerCounts[i].Count < answerCounts[j].Count
	})
	if len(answerCounts) == 0 {
		return "", true
	} else if len(answerCounts) != 1 && answerCounts[0].Count == answerCounts[1].Count {
		return "", true
	}
	i := slices.IndexFunc(poll.Answers, func(a discordgo.PollAnswer) bool {
		return a.AnswerID == answerCounts[0].ID
	})
	return poll.Answers[i].Media.Text, false
}

type SetPollChannelCommand struct {
	GuildID   string
	ChannelID string
}

func NewSetPollChannelCommand(guildID string, channel *discordgo.Channel) *SetPollChannelCommand {
	return &SetPollChannelCommand{
		GuildID:   guildID,
		ChannelID: channel.ID,
	}
}

func (c *SetPollChannelCommand) Execute(ctx context.Context, cl *clients.Clients) (*discordgo.WebhookEdit, error) {
	g, err := guild.GetGuild(ctx, c.GuildID, cl)
	if err != nil {
		return nil, fmt.Errorf("GetGuild: %v", err)
	}
	err = g.SetPollChannel(ctx, c.ChannelID)
	if err != nil {
		return nil, fmt.Errorf("SetPollChannel: %v", err)
	}
	return utils.NewWebhookEdit(fmt.Sprintf("Set poll channel to <#%v>", c.ChannelID)), nil
}
