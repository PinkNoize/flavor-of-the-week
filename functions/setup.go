package functions

import (
	"context"
	"encoding/hex"
	"log"
	"os"

	"cloud.google.com/go/pubsub"
	"go.uber.org/zap"
)

var projectID string = os.Getenv("PROJECT_ID")
var commandTopicID string = os.Getenv("COMMAND_TOPIC")

var discordPubkey []byte
var commandTopic *pubsub.Topic
var zapLogger *zap.Logger
var zapSlogger *zap.SugaredLogger

func init() {
	var err error
	ctx := context.Background()
	zapLogger, zapSlogger = setup_loggers()
	pubsubClient, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		zapSlogger.Fatalf("Failed to create pubsub client: %v", err)
	}
	commandTopic = pubsubClient.Topic(commandTopicID)

	discordPubkey, err = hex.DecodeString(os.Getenv("DISCORD_PUBKEY"))
	if err != nil {
		zapSlogger.Fatalf("Failed to decode public key: %v", err)
	}
}

func setup_loggers() (*zap.Logger, *zap.SugaredLogger) {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	slogger := logger.Sugar()
	return logger, slogger
}
