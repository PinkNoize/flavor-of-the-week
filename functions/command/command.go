package command

import (
	"context"
	"fmt"
	"io"

	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"github.com/PinkNoize/flavor-of-the-week/functions/utils"
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
	if c.interaction.Type == discordgo.InteractionApplicationCommand {
		ctx = context.WithValue(ctx, commandCtxKey, c.CommandName())
	}
	return ctx
}

func (c *DiscordCommand) LogCommand(ctx context.Context) {
	switch c.Type() {
	case discordgo.InteractionApplicationCommand, discordgo.InteractionApplicationCommandAutocomplete:
		command := c.interaction.ApplicationCommandData()

		verb := "ran"
		if c.Type() == discordgo.InteractionApplicationCommandAutocomplete {
			verb = "autocompleted"
		}

		ctxzap.Info(ctx, fmt.Sprintf("User %v (%v) %v %v", c.UserNick(), c.UserID(), verb, command.Name),
			zap.String("type", "audit"),
			zap.String("nick", c.UserNick()),
			zap.String("userid", c.UserID()),
			zap.String("command", command.Name),
			zap.String("guildID", c.interaction.GuildID),
			zap.Any("options", command.Options),
		)
	case discordgo.InteractionMessageComponent:
		data := c.interaction.MessageComponentData()

		ctxzap.Info(ctx, fmt.Sprintf("User %v (%v) interacted with a message", c.UserNick(), c.UserID()),
			zap.String("type", "audit"),
			zap.String("nick", c.UserNick()),
			zap.String("userid", c.UserID()),
			zap.String("custom_id", data.CustomID),
			zap.String("guildID", c.interaction.GuildID),
		)
	}
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
	switch c.Type() {
	case discordgo.InteractionApplicationCommand:
		return c.fromApplicationCommand()
	case discordgo.InteractionMessageComponent:
		return c.fromMessageComponent()
	}
	return nil, fmt.Errorf("Unexpected interaction type: %v", c.Type())
}

func (c *DiscordCommand) fromApplicationCommand() (Command, error) {
	if c.Type() != discordgo.InteractionApplicationCommand {
		return nil, fmt.Errorf("not a valid command")
	}
	commandData := c.interaction.ApplicationCommandData()
	args := utils.OptionsToMap(commandData.Options)
	switch commandData.Name {
	case "add":
		if pass, missing := utils.VerifyOpts(args, []string{"type", "name"}); !pass {
			return nil, fmt.Errorf("missing options: %v", missing)
		}
		return NewAddCommand(c.interaction.GuildID, args["type"].StringValue(), args["name"].StringValue()), nil
	case "remove":
		if pass, missing := utils.VerifyOpts(args, []string{"name"}); !pass {
			return nil, fmt.Errorf("missing options: %v", missing)
		}
		return NewRemoveCommand(c.interaction.GuildID, args["name"].StringValue(), false), nil
	case "nominations":
		subcmd := commandData.Options[0]
		subcmd_args := utils.OptionsToMap(subcmd.Options)
		switch subcmd.Name {
		case "add":
			if pass, missing := utils.VerifyOpts(subcmd_args, []string{"name"}); !pass {
				return nil, fmt.Errorf("missing options: %v", missing)
			}
			return NewNominationAddCommand(c.interaction.GuildID, c.UserID(), subcmd_args["name"].StringValue()), nil
		case "remove":
			if pass, missing := utils.VerifyOpts(subcmd_args, []string{"name"}); !pass {
				return nil, fmt.Errorf("missing options: %v", missing)
			}
			return NewNominationRemoveCommand(c.interaction.GuildID, c.UserID(), subcmd_args["name"].StringValue()), nil
		case "list":
			var name string
			nameOpt, ok := subcmd_args["name"]
			if ok {
				name = nameOpt.StringValue()
			}
			if c.interaction.Member == nil {
				return nil, fmt.Errorf("Member not found in interaction")
			}
			return NewNominationListCommand(c.interaction.GuildID, c.interaction.Member.User.ID, name, 0), nil
		default:
			return nil, fmt.Errorf("not a valid command: %v", subcmd.Name)
		}
	case "pool":
		var name string
		nameOpt, ok := args["name"]
		if ok {
			name = nameOpt.StringValue()
		}
		var actType string
		actTypeOpt, ok := args["type"]
		if ok {
			actType = actTypeOpt.StringValue()
		}
		return NewPoolListCommand(c.interaction.GuildID, name, actType, 0), nil
	case "start-poll":
		return NewStartPollCommand(c.interaction.GuildID), nil
	case "end-poll":
		return NewEndPollCommand(c.interaction.GuildID), nil
	case "poll-channel":
		if pass, missing := utils.VerifyOpts(args, []string{"channel"}); !pass {
			return nil, fmt.Errorf("missing options: %v", missing)
		}
		return NewSetPollChannelCommand(c.interaction.GuildID, args["channel"].ChannelValue(nil)), nil
	case "override-fow":
		if pass, missing := utils.VerifyOpts(args, []string{"name"}); !pass {
			return nil, fmt.Errorf("missing options: %v", missing)
		}
		return NewSetFowCommand(c.interaction.GuildID, args["name"].StringValue()), nil
	case "force-remove":
		if pass, missing := utils.VerifyOpts(args, []string{"name"}); !pass {
			return nil, fmt.Errorf("missing options: %v", missing)
		}
		return NewRemoveCommand(c.interaction.GuildID, args["name"].StringValue(), true), nil
	case "stats":
		return NewStatsCommand(c.interaction.GuildID), nil
	default:
		return nil, fmt.Errorf("not a valid command: %v", commandData.Name)
	}
}

func (c *DiscordCommand) fromMessageComponent() (Command, error) {
	if c.Type() != discordgo.InteractionMessageComponent {
		return nil, fmt.Errorf("not a valid message")
	}
	msgData := c.interaction.MessageComponentData()
	customID, err := utils.ParseCustomID(msgData.CustomID)
	if err != nil {
		return nil, err
	}
	switch msgData.ComponentType {
	case discordgo.ButtonComponent:
		switch customID.Type {
		case "pool":
			return NewPoolListCommand(c.interaction.GuildID, customID.Filter.Name, customID.Filter.Type, customID.Page), nil
		case "nominations":
			return NewNominationListCommand(c.interaction.GuildID, c.interaction.Member.User.ID, customID.Filter.Name, customID.Page), nil
		}
	}
	return nil, fmt.Errorf("Unexpected message component: %v", msgData)
}

type Command interface {
	Execute(ctx context.Context, cl *clients.Clients) (*discordgo.WebhookEdit, error)
}
