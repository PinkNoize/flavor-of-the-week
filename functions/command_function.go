package functions

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/PinkNoize/flavor-of-the-week/functions/command"
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

	cmd, err := command.FromReader(ctx, bytes.NewReader(m.Data))
	if err != nil {
		return fmt.Errorf("error parsing command: %v", err)
	}
	cmd.LogCommand(ctx)

	return nil
}
