package functions

import (
	"context"
	"encoding/hex"
	"log"
	"os"

	"cloud.google.com/go/pubsub"
	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var projectID string = os.Getenv("PROJECT_ID")
var commandTopicID string = os.Getenv("COMMAND_TOPIC")

var discordPubkey []byte
var clientLoader *clients.Clients
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
	clientLoader = clients.New(ctx, projectID)
}

func setup_loggers() (*zap.Logger, *zap.SugaredLogger) {
	logger, err := newZapLogger()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	slogger := logger.Sugar()
	return logger, slogger
}

func newZapLogger() (*zap.Logger, error) {
	loggerCfg := &zap.Config{
		Level:            zap.NewAtomicLevelAt(zapcore.InfoLevel),
		Encoding:         "json",
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	plain, err := loggerCfg.Build(zap.AddStacktrace(zap.DPanicLevel))
	if err != nil {
		return nil, err
	}
	return plain, nil
}

var encoderConfig = zapcore.EncoderConfig{
	TimeKey:        "time",
	LevelKey:       "severity",
	NameKey:        "logger",
	CallerKey:      "caller",
	MessageKey:     "message",
	StacktraceKey:  "stacktrace",
	LineEnding:     zapcore.DefaultLineEnding,
	EncodeLevel:    encodeLevel(),
	EncodeTime:     zapcore.ISO8601TimeEncoder,
	EncodeDuration: zapcore.MillisDurationEncoder,
	EncodeCaller:   zapcore.ShortCallerEncoder,
}

func encodeLevel() zapcore.LevelEncoder {
	return func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
		switch l {
		case zapcore.DebugLevel:
			enc.AppendString("DEBUG")
		case zapcore.InfoLevel:
			enc.AppendString("INFO")
		case zapcore.WarnLevel:
			enc.AppendString("WARNING")
		case zapcore.ErrorLevel:
			enc.AppendString("ERROR")
		case zapcore.DPanicLevel:
			enc.AppendString("CRITICAL")
		case zapcore.PanicLevel:
			enc.AppendString("ALERT")
		case zapcore.FatalLevel:
			enc.AppendString("EMERGENCY")
		}
	}
}
