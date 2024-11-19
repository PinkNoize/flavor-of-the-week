package command

import (
        "cmp"
        "context"
        "fmt"
        "net/http"
        "slices"
        "time"

        "math/rand"

        "github.com/PinkNoize/flavor-of-the-week/functions/activity"
        "github.com/PinkNoize/flavor-of-the-week/functions/clients"
        "github.com/PinkNoize/flavor-of-the-week/functions/guild"
        "github.com/PinkNoize/flavor-of-the-week/functions/utils"
        "github.com/bwmarrin/discordgo"
        "github.com/cenkalti/backoff/v4"
        "github.com/elliotchance/orderedmap/v2"
        "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
)

const MAX_POLL_ENTRIES int = 7

type CreatePollCommand struct {
        GuildID             string
        Options             []discordgo.PollAnswer
        Duration            int
        SuddenDeath         bool
        skipActivePollCheck bool
}

func NewCreatePollCommand(guildID string, options []discordgo.PollAnswer, duration int, suddenDeath bool) *CreatePollCommand {
        return &CreatePollCommand{
                GuildID:     guildID,
                Options:     options,
                Duration:    duration,
                SuddenDeath: suddenDeath,
        }
}

func (c *CreatePollCommand) SkipActivePollCheck(skip bool) {
        c.skipActivePollCheck = skip
}

func (c *CreatePollCommand) Execute(ctx context.Context, cl *clients.Clients) (*discordgo.WebhookEdit, error) {
        s, err := cl.Discord()
        if err != nil {
                return nil, fmt.Errorf("Discord: %v", err)
        }
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
        if !c.skipActivePollCheck && pollID != nil {
                return utils.NewWebhookEdit("There is already an active poll"), nil
        }
        if c.Options == nil {
                c.Options, err = GeneratePollEntries(ctx, g, cl)
                if err != nil {
                        return nil, fmt.Errorf("GeneratePollEntries: %v", err)
                }
        }

        text := "What should the flavor of the week be?"
        if c.SuddenDeath {
                text = "Sudden Death Tie Breaker"
        }

        msg, err := s.ChannelMessageSendComplex(*chanID, &discordgo.MessageSend{
                Poll: &discordgo.Poll{
                        Question: discordgo.PollMedia{
                                Text: text,
                        },
                        Answers:          c.Options,
                        AllowMultiselect: true,
                        LayoutType:       discordgo.PollLayoutTypeDefault,
                        Duration:         c.Duration,
                },
        })
        if err != nil {
                return nil, fmt.Errorf("ChannelMessageSendComplex: %v", err)
        }
        err = g.SetActivePoll(ctx, &guild.PollInfo{
                ChannelID:   *chanID,
                MessageID:   msg.ID,
                SuddenDeath: c.SuddenDeath,
        })
        if err != nil {
                return nil, fmt.Errorf("SetActivePoll: %v", err)
        }
        msgLink := fmt.Sprintf("https://discord.com/channels/%v/%v/%v", c.GuildID, *chanID, msg.ID)
        return utils.NewWebhookEdit(fmt.Sprintf("Poll created: %v", msgLink)), nil
}

type StartPollCommand struct {
        GuildID             string
        skipActivePollCheck bool
}

func NewStartPollCommand(guildID string) *StartPollCommand {
        return &StartPollCommand{
                GuildID: guildID,
        }
}

func (c *StartPollCommand) SkipActivePollCheck(skip bool) {
        c.skipActivePollCheck = skip
}

func (c *StartPollCommand) Execute(ctx context.Context, cl *clients.Clients) (*discordgo.WebhookEdit, error) {
        pollCmd := NewCreatePollCommand(
                c.GuildID,
                nil,
                48,
                false,
        )
        pollCmd.SkipActivePollCheck(c.skipActivePollCheck)
        return pollCmd.Execute(ctx, cl)

}

func GeneratePollEntries(ctx context.Context, guild *guild.Guild, cl *clients.Clients) ([]discordgo.PollAnswer, error) {
        type answerEntry struct {
                count int
                emoji string
        }

        answers := orderedmap.NewOrderedMap[string, answerEntry]()

        fow, err := guild.GetFow(ctx)
        if err != nil {
                return nil, fmt.Errorf("GetFow: %v", err)
        }
        if fow != nil {
                answers.Set(*fow, answerEntry{
                        count: 1,
                        emoji: "📌",
                })
        }

        ctxzap.Info(ctx, "Getting top nominations")
        // Add top nominations. Add one in case the current FOW is a top nomination
        nominations, err := activity.GetTopNominations(ctx, guild.GetGuildId(), MAX_POLL_ENTRIES-answers.Len()+1, cl)
        if err != nil {
                return nil, fmt.Errorf("GetTopNominations: %v", err)
        }

        for _, nom := range nominations {
                tmp := answers.GetOrDefault(nom, answerEntry{
                        count: 0,
                        emoji: "🗳️",
                })
                tmp.count += 1
                answers.Set(nom,
                        tmp,
                )
                if answers.Len() == MAX_POLL_ENTRIES {
                        break
                }
        }
        // Now pick random entries
        loop_count := 0
out:
        for answers.Len() < MAX_POLL_ENTRIES && loop_count < 5 {
                ctxzap.Info(ctx, fmt.Sprintf("Getting random activities nominations. Try %v", loop_count))
                randomsChoices, err := activity.GetRandomActivities(ctx, guild.GetGuildId(), MAX_POLL_ENTRIES-answers.Len(), cl)
                if err != nil {
                        return nil, fmt.Errorf("GetRandomActivities: %v", err)
                }
                for _, choice := range randomsChoices {
                        tmp := answers.GetOrDefault(choice, answerEntry{
                                count: 0,
                                emoji: "🎰",
                        })
                        tmp.count += 1
                        answers.Set(choice,
                                tmp,
                        )
                }

                // Check if we are repeating which is indicative of not enough answers in the pool to fill a poll
                for el := answers.Back(); el != nil; el = el.Prev() {
                        if el.Value.count > 5 {
                                break out
                        }
                }
                loop_count += 1
        }

        ctxzap.Info(ctx, fmt.Sprintf("Generated poll entries: %v", answers))
        results := make([]discordgo.PollAnswer, 0, answers.Len())
        for el := answers.Front(); el != nil; el = el.Next() {
                results = append(results, discordgo.PollAnswer{
                        Media: &discordgo.PollMedia{
                                Text: truncateActivityName(el.Key),
                                Emoji: &discordgo.ComponentEmoji{
                                        Name: el.Value.emoji,
                                },
                        },
                })
        }
        results = append(results, discordgo.PollAnswer{
                Media: &discordgo.PollMedia{
                        Text: "Reroll",
                        Emoji: &discordgo.ComponentEmoji{
                                Name: "🎲",
                        },
                },
        })
        return results, nil
}

func truncateActivityName(name string) string {
        if len(name) > 55 {
                return fmt.Sprintf("%v...", name[:52])
        }
        return name
}

func recoverTruncatedActivity(ctx context.Context, name, guildID string, cl *clients.Clients) (string, error) {
        if len(name) == 55 && name[52:] == "..." {
                // Name may be truncated
                fullName, err := activity.RecoverActivity(ctx, guildID, name[:52], cl)
                if err != nil {
                        return "", fmt.Errorf("RecoverActivity: %v", err)
                }
                return fullName, nil
        } else {
                return name, nil
        }

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
        // Get the poll status
        msg, err := s.ChannelMessage(pollID.ChannelID, pollID.MessageID)
        if err != nil {
                if restErr, ok := err.(*discordgo.RESTError); ok && restErr.Response.StatusCode == http.StatusNotFound {
                        // If the poll has been deleted, reset the poll status
                        err := g.ClearActivePoll(ctx)
                        if err != nil {
                                return nil, fmt.Errorf("ClearActivePoll: %v", err)
                        }
                }
                return utils.NewWebhookEdit("⚠️ Unable to retrieve the poll"), fmt.Errorf("ChannelMessage: %v", err)
        }
        if msg.Poll == nil {
                return utils.NewWebhookEdit("⚠️ No poll associated with the message"), fmt.Errorf("Missing poll")
        }
        if msg.Poll.Results == nil || !msg.Poll.Results.Finalized || msg.Poll.Results.AnswerCounts == nil {
                msg, err = s.PollExpire(pollID.ChannelID, pollID.MessageID)
                if err != nil {
                        return utils.NewWebhookEdit("⚠️ Unable to end the poll"), fmt.Errorf("PollExpire: %v", err)
                }
                waitForResults := func() error {
                        msg, err = s.ChannelMessage(pollID.ChannelID, pollID.MessageID)
                        if err != nil || msg.Poll == nil {
                                return fmt.Errorf("ChannelMessage: %v", err)
                        }
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
        // Poll has ended, get the results
        winners, tie := determinePollWinners(msg.Poll)
        var response *discordgo.WebhookEdit
        if tie {
                if pollID.SuddenDeath {
                        // If it is a sudden death poll, choose at random
                        winner := winners[rand.Intn(len(winners))]
                        // TODO: Post winner and cleanup
                        s.ChannelMessageSendComplex(pollID.ChannelID, &discordgo.MessageSend{
                                Embeds: []*discordgo.MessageEmbed{
                                        {
                                                Title:       "⚡Sudden Death Winner⚡",
                                                Description: fmt.Sprintf("This winner is {}", winner),
                                        },
                                },
                        })
                } else {
                        // Start a sudden death poll
                        pollWinners := make([]discordgo.PollAnswer, 0)
                        for _, ans := range winners {
                                pollWinners = append(pollWinners, discordgo.PollAnswer{
                                        Media: &discordgo.PollMedia{
                                                Text: ans,
                                                Emoji: &discordgo.ComponentEmoji{
                                                        Name: "⚡",
                                                },
                                        },
                                })
                        }
                        pollCmd := NewCreatePollCommand(c.GuildID, pollWinners, 2, true)
                        pollCmd.SkipActivePollCheck(true)
                        return pollCmd.Execute(ctx, cl)
                }
                response = utils.NewWebhookEdit("Ended the poll with a tie")
        } else if winners[0] == "Reroll" {
                // Create a new poll
                pollCmd := NewStartPollCommand(c.GuildID)
                pollCmd.SkipActivePollCheck(true)
                return pollCmd.Execute(ctx, cl)
        } else {
                // Recover truncated name
                winner, err := recoverTruncatedActivity(ctx, winners[0], c.GuildID, cl)
                if err != nil {
                        return nil, fmt.Errorf("recoverTruncatedActivity: %v", err)
                }
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

func determinePollWinners(poll *discordgo.Poll) ([]string, bool) {
        answerCounts := poll.Results.AnswerCounts
        // There are no votes
        if len(answerCounts) == 0 {
                return nil, true
        }

        // sort results
        slices.SortFunc(answerCounts, func(a, b *discordgo.PollAnswerCount) int {
                if a == nil && b != nil {
                        return 1
                } else if a != nil && b == nil {
                        return -1
                } else if a == nil && b == nil {
                        return 0
                }
                return -cmp.Compare(a.Count, b.Count)
        })
        maxVote := answerCounts[0].Count

        winners := make([]string, 0)
        for _, ans := range answerCounts {
                if ans.Count == maxVote {
                        i := slices.IndexFunc(poll.Answers, func(a discordgo.PollAnswer) bool {
                                return a.AnswerID == ans.ID
                        })
                        winners = append(winners, poll.Answers[i].Media.Text)
                } else {
                        break
                }
        }
        return winners, len(winners) > 1
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

type SchedulePollCommand struct {
        GuildID string
        Day     string
        Hour    int
}

func NewSchedulePollCommand(guildID, day string, hour int) *SchedulePollCommand {
        return &SchedulePollCommand{
                GuildID: guildID,
                Day:     day,
                Hour:    hour,
        }
}

func (c *SchedulePollCommand) Execute(ctx context.Context, cl *clients.Clients) (*discordgo.WebhookEdit, error) {
        g, err := guild.GetGuild(ctx, c.GuildID, cl)
        if err != nil {
                return nil, fmt.Errorf("GetGuild: %v", err)
        }

        dayLookup := map[string]time.Weekday{
                "Sunday":    time.Sunday,
                "Monday":    time.Monday,
                "Tuesday":   time.Tuesday,
                "Wednesday": time.Wednesday,
                "Thursday":  time.Thursday,
                "Friday":    time.Friday,
                "Saturday":  time.Saturday,
        }
        day := dayLookup[c.Day]

        err = g.SetSchedule(ctx, &guild.ScheduleInfo{
                Day:  day,
                Hour: c.Hour,
        })
        if err != nil {
                return nil, fmt.Errorf("SetSchedule: %v", err)
        }
        return utils.NewWebhookEdit(fmt.Sprintf("Set schedule for every %v on %v", c.Day, c.Hour)), nil
}
