package guild

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"github.com/PinkNoize/flavor-of-the-week/functions/setup"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
)

type PollInfo struct {
	ChannelID string `firestore:"channel_id"`
	MessageID string `firestore:"message_id"`
}

type innerGuild struct {
	PollChannelID *string   `firestore:"poll_channel_id"`
	ActivePoll    *PollInfo `firestore:"active_poll"`
	Fow           *string   `firestore:"fow"`
	FowCount      int       `firestore:"fow_count"`
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
	return firestoreClient.Collection(fmt.Sprintf("flavor-of-the-week-guilds-%v", setup.ENV)), nil
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

func (g *Guild) GetGuildId() string {
	return g.docRef.ID
}

func (g *Guild) load(ctx context.Context) error {
	if !g.loaded {
		snap, err := g.docRef.Get(ctx)
		if err != nil {
			return err
		}
		err = snap.DataTo(&g.inner)
		if err != nil {
			return fmt.Errorf("DataTo: %v", err)
		}
		g.loaded = true
	}
	return nil
}

func (g *Guild) SetPollChannel(ctx context.Context, channelId string) error {
	_, err := g.docRef.Set(ctx, map[string]interface{}{
		"poll_channel_id": channelId,
	}, firestore.MergeAll)
	if err != nil {
		return err
	}
	g.inner.PollChannelID = &channelId
	return nil
}

func (g *Guild) GetPollChannel(ctx context.Context) (*string, error) {
	err := g.load(ctx)
	if err != nil {
		return nil, err
	}
	return g.inner.PollChannelID, nil
}

func (g *Guild) SetFow(ctx context.Context, fow string) error {
	_, err := g.docRef.Set(ctx, map[string]interface{}{
		"fow":       fow,
		"fow_count": firestore.Increment(1),
	}, firestore.MergeAll)
	if err != nil {
		return err
	}
	g.inner.Fow = &fow
	return nil
}

func (g *Guild) GetFow(ctx context.Context) (*string, error) {
	err := g.load(ctx)
	if err != nil {
		return nil, err
	}
	return g.inner.Fow, nil
}

func (g *Guild) GetFowCount(ctx context.Context) (int, error) {
	err := g.load(ctx)
	if err != nil {
		return 0, err
	}
	return g.inner.FowCount, nil
}

func (g *Guild) GetActivePoll(ctx context.Context) (*PollInfo, error) {
	err := g.load(ctx)
	if err != nil {
		return nil, err
	}
	return g.inner.ActivePoll, nil
}

func (g *Guild) ClearActivePoll(ctx context.Context) error {
	ctxzap.Info(ctx, "Clearing active poll")
	_, err := g.docRef.Update(ctx, []firestore.Update{
		{
			Path:  "active_poll",
			Value: firestore.Delete,
		},
	})
	if err != nil {
		return err
	}
	g.inner.ActivePoll = nil
	return nil
}

func (g *Guild) SetActivePoll(ctx context.Context, pollInfo *PollInfo) error {
	_, err := g.docRef.Set(ctx, map[string]interface{}{
		"active_poll": pollInfo,
	}, firestore.MergeAll)
	if err != nil {
		return err
	}
	g.inner.ActivePoll = pollInfo
	return nil
}
