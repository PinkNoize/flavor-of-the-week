package command

import (
	"context"
	"fmt"
	"io"

	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"github.com/bwmarrin/discordgo"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

type contextKey string

const (
	nickCtxKey    contextKey = "nick"
	userIdCtxKey  contextKey = "userid"
	commandCtxKey contextKey = "command"
)

type DiscordCommand struct {
	interaction    discordgo.Interaction
	rawInteraction []byte
}

func FromReader(ctx context.Context, r io.Reader) (DiscordCommand, error) {
	var command DiscordCommand
	var err error
	command.rawInteraction, err = io.ReadAll(r)
	if err != nil {
		return command, err
	}
	err = command.interaction.UnmarshalJSON(command.rawInteraction)
	return command, err
}

func (c *DiscordCommand) ToContext(ctx context.Context) context.Context {
	ctx = context.WithValue(ctx, nickCtxKey, c.UserNick())
	ctx = context.WithValue(ctx, userIdCtxKey, c.UserID())
	ctx = context.WithValue(ctx, commandCtxKey, c.CommandName())
	return ctx
}

func (c *DiscordCommand) LogCommand(ctx context.Context) {
	command := c.interaction.ApplicationCommandData()

	ctxzap.Info(ctx, fmt.Sprintf("User %v (%v) ran %v", c.UserNick(), c.UserID(), command.Name),
		zap.String("type", "audit"),
		zap.String("nick", c.UserNick()),
		zap.String("userid", c.UserID()),
		zap.String("command", command.Name),
		zap.Any("options", command.Options),
	)
}

func (c *DiscordCommand) Type() discordgo.InteractionType {
	return c.interaction.Type
}

func (c *DiscordCommand) Interaction() *discordgo.Interaction {
	return &c.interaction
}

func (c *DiscordCommand) RawInteraction() []byte {
	return c.rawInteraction
}

func (c *DiscordCommand) UserID() string {
	if c.interaction.Member != nil {
		return c.interaction.Member.User.ID
	} else if c.interaction.User != nil {
		return c.interaction.User.ID
	} else {
		return ""
	}
}

func (c *DiscordCommand) UserNick() string {
	if c.interaction.Member != nil {
		name := c.interaction.Member.Nick
		if name == "" {
			name = fmt.Sprintf("%v%v", c.interaction.Member.User.ID, c.interaction.Member.User.Discriminator)
		}
		return name
	} else if c.interaction.User != nil {
		return fmt.Sprintf("%v%v", c.interaction.User.ID, c.interaction.User.Discriminator)
	} else {
		return ""
	}
}

func (c *DiscordCommand) CommandName() string {
	return c.interaction.ApplicationCommandData().Name
}

func (c *DiscordCommand) ToCommand() (Command, error) {
	if c.Type() != discordgo.InteractionApplicationCommand {
		return nil, fmt.Errorf("not a valid command")
	}
	commandData := c.interaction.ApplicationCommandData()
	args := optionsToMap(commandData.Options)
	switch commandData.Name {
	case "add":
		if pass, missing := verifyOpts(args, []string{"type", "name"}); !pass {
			return nil, fmt.Errorf("missing options: %v", missing)
		}
		return NewAddCommand(c.interaction.GuildID, args["type"].StringValue(), args["name"].StringValue()), nil
	case "remove":
		if pass, missing := verifyOpts(args, []string{"name"}); !pass {
			return nil, fmt.Errorf("missing options: %v", missing)
		}
		return NewRemoveCommand(c.interaction.GuildID, args["name"].StringValue()), nil
	case "nominations":
		subcmd := commandData.Options[0]
		subcmd_args := optionsToMap(subcmd.Options)
		switch subcmd.Name {
		case "add":
			if pass, missing := verifyOpts(subcmd_args, []string{"name"}); !pass {
				return nil, fmt.Errorf("missing options: %v", missing)
			}
			return NewNominationAddCommand(c.interaction.GuildID, subcmd_args["name"].StringValue()), nil
		case "remove":
			if pass, missing := verifyOpts(subcmd_args, []string{"name"}); !pass {
				return nil, fmt.Errorf("missing options: %v", missing)
			}
			return NewNominationAddCommand(c.interaction.GuildID, subcmd_args["name"].StringValue()), nil
		case "list":
			var name string
			nameOpt, ok := args["name"]
			if !ok {
				name = nameOpt.StringValue()
			}
			return NewNominationListCommand(c.interaction.GuildID, c.interaction.User.ID, name), nil
		default:
			return nil, fmt.Errorf("not a valid command: %v", subcmd.Name)
		}
	case "pool":
		var name string
		nameOpt, ok := args["name"]
		if !ok {
			name = nameOpt.StringValue()
		}
		var actType string
		actTypeOpt, ok := args["type"]
		if !ok {
			actType = actTypeOpt.StringValue()
		}
		return NewPoolListCommand(c.interaction.GuildID, name, actType), nil
	case "start-poll":
		return NewStartPollCommand(c.interaction.GuildID), nil
	case "poll-channel":
		if pass, missing := verifyOpts(args, []string{"channel"}); !pass {
			return nil, fmt.Errorf("missing options: %v", missing)
		}
		return NewSetPollChannelCommand(c.interaction.GuildID, args["channel"].ChannelValue(nil)), nil
	default:
		return nil, fmt.Errorf("not a valid command: %v", commandData.Name)
	}
}

func optionsToMap(opts []*discordgo.ApplicationCommandInteractionDataOption) map[string]*discordgo.ApplicationCommandInteractionDataOption {
	mappedOpts := make(map[string]*discordgo.ApplicationCommandInteractionDataOption)

	for i := range opts {
		if opts[i] != nil {
			mappedOpts[opts[i].Name] = opts[i]
		}
	}
	return mappedOpts
}

func verifyOpts(opts map[string]*discordgo.ApplicationCommandInteractionDataOption, expected []string) (bool, string) {
	for _, v := range expected {
		if _, ok := opts[v]; !ok {
			return false, v
		}
	}
	return true, ""
}

type Command interface {
	Execute(ctx context.Context, cl *clients.Clients) (*discordgo.WebhookParams, error)
}
