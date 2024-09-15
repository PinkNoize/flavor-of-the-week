package functions

import (
	"encoding/hex"
	"log"
	"os"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

var logger *zap.Logger
var slogger *zap.SugaredLogger
var discordPubkey []byte
var discordSession *discordgo.Session

func init() {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	slogger = logger.Sugar()

	discordPubkey, err = hex.DecodeString(os.Getenv("DISCORD_PUBKEY"))
	if err != nil {
		slogger.Fatalf("Failed to decode public key: %v", err)
	}
	discordSession, err = discordgo.New("")
	if err != nil {
		slogger.Fatalf("Error: initDiscord: %v", err)
	}
}
