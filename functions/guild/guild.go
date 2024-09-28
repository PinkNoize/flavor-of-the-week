package guild

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
)

type innerGuild struct {
	PollChannelID *string `firestore:"poll_channel_id"`
}

type Guild struct {
	inner  innerGuild
	loaded bool
	docRef *firestore.DocumentRef
}

func getCollection(cl *clients.Clients) (*firestore.CollectionRef, error) {
	firestoreClient, err := cl.Firestore()
	if err != nil {
		return nil, err
	}
	return firestoreClient.Collection("flavor-of-the-week-guilds"), nil
}

func generateName(guildID string) string {
	return guildID
}

func GetGuild(ctx context.Context, guildID string, cl *clients.Clients) (*Guild, error) {
	activityCollection, err := getCollection(cl)
	if err != nil {
		return nil, fmt.Errorf("getCollection: %v", err)
	}
	docName := generateName(guildID)
	guildDoc := activityCollection.Doc(docName)
	return &Guild{
		docRef: guildDoc,
		loaded: false,
	}, nil
}

func (g *Guild) SetPollChannel(ctx context.Context, channelId string) error {
	_, err := g.docRef.Update(ctx, []firestore.Update{
		{
			Path:  "poll_channel_id",
			Value: channelId,
		},
	})
	if err != nil {
		return err
	}
	g.inner.PollChannelID = &channelId
	return nil
}
