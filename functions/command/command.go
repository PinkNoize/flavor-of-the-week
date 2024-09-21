package command

import (
	"context"
	"fmt"
	"io"

	"github.com/bwmarrin/discordgo"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
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
	l := ctxzap.Extract(ctx)
	command := c.interaction.ApplicationCommandData()

	l.Sugar().Infow("audit",
		"nick", c.UserNick(),
		"userid", c.UserID(),
		"command", command.Name,
		"options", command.Options,
	)
}

func (c *DiscordCommand) Type() discordgo.InteractionType {
	return c.interaction.Type
}

func (c *DiscordCommand) Interaction() discordgo.Interaction {
	return c.interaction
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
