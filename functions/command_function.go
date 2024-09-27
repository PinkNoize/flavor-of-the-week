package functions

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/PinkNoize/flavor-of-the-week/functions/command"
	"github.com/bwmarrin/discordgo"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
)

// PubSubMessage is the payload of a Pub/Sub event.
// See the documentation for more details:
// https://cloud.google.com/pubsub/docs/reference/rest/v1/PubsubMessage
type PubSubMessage struct {
	Data       []byte          `json:"data"`
	Attributes json.RawMessage `json:"attributes"`
}

func CommandPubSub(ctx context.Context, m PubSubMessage) error {
	var err error
	logger, slogger := zapLogger, zapSlogger
	defer func() {
		err = errors.Join(slogger.Sync())
		err = errors.Join(logger.Sync())
	}()
	ctx = ctxzap.ToContext(ctx, logger)

	discordCmd, err := command.FromReader(ctx, bytes.NewReader(m.Data))
	if err != nil {
		return fmt.Errorf("error parsing command: %v", err)
	}
	discordCmd.LogCommand(ctx)
	ctx = discordCmd.ToContext(ctx)

	var response *discordgo.InteractionResponse = nil
	defer func() {
		ctxzap.Info(ctx, "Sending Interaction response")
		discordSession, err := clientLoader.Discord()
		if err != nil {
			ctxzap.Error(ctx, fmt.Sprintf("Failed to initalize discord client: %v", err))
			return
		}
		if response == nil {
			response = &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Internal Error",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			}
		}
		err = discordSession.InteractionRespond(discordCmd.Interaction(), response)
		if err != nil {
			ctxzap.Error(ctx, fmt.Sprintf("InteractionRespond %v", err))
			return
		}
	}()

	cmd, err := discordCmd.ToCommand()
	if err != nil {
		return fmt.Errorf("converting to command: %v", err)
	}
	response, err = cmd.Execute(ctx, clientLoader)
	if err != nil {
		return fmt.Errorf("executing command: %v", err)
	}

	return nil
}
