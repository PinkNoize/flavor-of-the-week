package functions

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"cloud.google.com/go/pubsub"

	"github.com/PinkNoize/flavor-of-the-week/functions/command"
	"github.com/bwmarrin/discordgo"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
)

func DiscordFunctionEntry(w http.ResponseWriter, r *http.Request) {
	var err error
	logger, slogger := zapLogger, zapSlogger
	defer func() {
		err = errors.Join(slogger.Sync())
		err = errors.Join(logger.Sync())
	}()
	ctx := ctxzap.ToContext(r.Context(), logger)

	verified := discordgo.VerifyInteraction(r, ed25519.PublicKey(discordPubkey))
	if !verified {
		slogger.Infow("Failed signature verification",
			"IP", r.RemoteAddr,
			"url", r.URL.Path,
		)
		http.Error(w, "signature mismatch", http.StatusUnauthorized)
		return
	}
	defer r.Body.Close()

	cmd, err := command.FromReader(ctx, r.Body)
	if err != nil {
		slogger.Errorf("Error parsing command: %v", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Log command
	cmd.LogCommand(ctx)
	// Add command context to ctx
	ctx = cmd.ToContext(ctx)

	switch cmd.Type() {
	case discordgo.InteractionPing:
		handlePing(ctx, w)
	case discordgo.InteractionApplicationCommand:
		err = forwardCommand(ctx, &cmd)
		if err != nil {
			slogger.Errorw("Failed to forward command",
				"error", err,
			)
			return
		}
		slogger.Info("Deferring response...")
		err = writeDeferredResponse(w, discordgo.InteractionResponseDeferredChannelMessageWithSource)
		if err != nil {
			slogger.Errorw("Failed to return deferred response",
				"error", err,
			)
			return
		}
	case discordgo.InteractionMessageComponent:
		slogger.Info("Deferring response...")
		err = writeDeferredResponse(w, discordgo.InteractionResponseDeferredMessageUpdate)
		if err != nil {
			slogger.Errorw("Failed to return deferred response",
				"error", err,
			)
			return
		}
	case discordgo.InteractionApplicationCommandAutocomplete:
		slogger.Error("Autocomplete not implemented")
		http.Error(w, "Autocomplete not implemented", http.StatusNotImplemented)
	default:
		slogger.Errorw("Unknown Interaction Type",
			"interactionType", cmd.Type(),
		)
		http.Error(w, "Unknown Interaction Type", http.StatusNotImplemented)
	}
}

func handlePing(ctx context.Context, w http.ResponseWriter) {
	l := ctxzap.Extract(ctx)
	l.Info("Ping received")
	_, err := w.Write([]byte(`{"type":1}`))
	if err != nil {
		l.Sugar().Errorw("Failed to write ping",
			"error", err,
		)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func forwardCommand(ctx context.Context, command *command.DiscordCommand) error {
	result := commandTopic.Publish(ctx, &pubsub.Message{
		Data: command.RawInteraction(),
	})
	_, err := result.Get(ctx)
	if err != nil {
		return fmt.Errorf("Pubsub.Publish: %v", err)
	}
	return nil
}

func writeDeferredResponse(w http.ResponseWriter, typ discordgo.InteractionResponseType) error {
	response := discordgo.InteractionResponse{
		Type: typ, // Deferred response
		Data: &discordgo.InteractionResponseData{
			Content: "...",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}

	// MUST SET HEADER BEFORE CONTENT
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		return fmt.Errorf("writeDeferredResponse: jsonEncoder: %v", err)
	}
	return nil
}
