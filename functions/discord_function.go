package functions

import (
	"context"
	"crypto/ed25519"
	"io"
	"net/http"

	"github.com/bwmarrin/discordgo"
)

func DiscordFunctionEntry(w http.ResponseWriter, r *http.Request) {
	defer logger.Sync()

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

	var interaction discordgo.Interaction
	rawInteraction, err := io.ReadAll(r.Body)
	if err != nil {
		slogger.Error("failed to ReadAll",
			"error", err,
		)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	err = interaction.UnmarshalJSON(rawInteraction)
	if err != nil {
		slogger.Error("failed to unmarshal interaction",
			"error", err,
		)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	switch interaction.Type {
	case discordgo.InteractionPing:
		handlePing(w)
	case discordgo.InteractionApplicationCommand:
		handleApplicationCommand(r.Context(), &interaction, w, rawInteraction)
	default:
		slogger.Error("Unknown Interaction Type",
			"interactionType", interaction.Type,
		)
		http.Error(w, "Unknown Interaction Type", http.StatusNotImplemented)
	}
}

func handlePing(w http.ResponseWriter) {
	slogger.Info("Ping received")
	_, err := w.Write([]byte(`{"type":1}`))
	if err != nil {
		slogger.Error("Failed to write ping",
			"error", err,
		)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func handleApplicationCommand(_ context.Context, interaction *discordgo.Interaction, w http.ResponseWriter, _ []byte) {
	slogger.Error("Unknown Interaction Type",
		"interactionType", interaction.Type,
	)
	http.Error(w, "Unknown Interaction Type", http.StatusNotImplemented)
}
