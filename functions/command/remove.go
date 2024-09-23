package command

import (
	"context"
	"crypto/sha256"
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/PinkNoize/flavor-of-the-week/functions/activity"
	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"github.com/bwmarrin/discordgo"
)

type RemoveCommand struct {
	GuildID string
	Name    string
}

func NewRemoveCommand(guildID, name string) *RemoveCommand {
	return &RemoveCommand{
		GuildID: guildID,
		Name:    name,
	}
}

func (c *RemoveCommand) Execute(ctx context.Context, cl *clients.Clients) (*discordgo.WebhookParams, error) {
	firestoreClient, err := cl.Firestore()
	if err != nil {
		return nil, err
	}
	docName := fmt.Sprintf("%x", sha256.Sum256([]byte(c.Name)))
	var act activity.Activity
	activityDoc := firestoreClient.Collection(c.GuildID).Doc(docName)
	activityDocSnap, err := activityDoc.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get Doc %v: %v", docName, err)
	}
	if !activityDocSnap.Exists() {
		return &discordgo.WebhookParams{
			Content: fmt.Sprintf("%v does not exist", c.Name),
			Flags:   discordgo.MessageFlagsEphemeral,
		}, nil
	}
	err = activityDocSnap.DataTo(&act)
	if err != nil {
		return nil, fmt.Errorf("Failed to deserialize activity: %v", err)
	}
	if len(act.Nominations) > 0 {
		return &discordgo.WebhookParams{
			Content: fmt.Sprintf("Cannot remove %v as it has nominations", c.Name),
			Flags:   discordgo.MessageFlagsEphemeral,
		}, nil
	}
	_, err = activityDoc.Delete(ctx, firestore.LastUpdateTime(activityDocSnap.UpdateTime))
	if err != nil {
		return &discordgo.WebhookParams{
			Content: fmt.Sprintf("Failed to remove %v from the pool. It may have been nominated while removing. Please try again", c.Name),
			Flags:   discordgo.MessageFlagsEphemeral,
		}, fmt.Errorf("Failed to delete %v (%v)", docName, c.Name)
	}
	return &discordgo.WebhookParams{
		Content: fmt.Sprintf("Removed %v from the pool", c.Name),
		Flags:   discordgo.MessageFlagsEphemeral,
	}, nil
}
