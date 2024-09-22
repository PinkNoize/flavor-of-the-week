package command

import (
	"context"
	"fmt"

	"github.com/PinkNoize/flavor-of-the-week/functions/activity"
	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AddCommand struct {
	GuildID      string
	ActivityType string
	Name         string
}

func NewAddCommand(guildID, activityType, name string) *AddCommand {
	return &AddCommand{
		GuildID:      guildID,
		ActivityType: activityType,
		Name:         name,
	}
}

func (c *AddCommand) Execute(ctx context.Context, cl *clients.Clients) error {
	firestoreClient, err := cl.Firestore()
	if err != nil {
		return err
	}
	var act *activity.Activity
	activityDoc := firestoreClient.Collection(c.GuildID).Doc(c.Name)
	switch c.ActivityType {
	case "activity":
		act = activity.NewActivity(activity.ACTIVITY, c.Name)
	// TODO: check activity type, if game lookup, validate first
	default:
		return fmt.Errorf("Activity type not supported: %v", c.ActivityType)
	}
	_, err = activityDoc.Create(ctx, act)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return fmt.Errorf("activity %v already exists", c.Name)
		}
		return fmt.Errorf("activityDoc.Create: %v", err)
	}
	return nil
}
