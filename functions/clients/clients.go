package clients

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/bwmarrin/discordgo"
	"github.com/josestg/lazy"
)

type Clients struct {
	Ctx             context.Context
	ProjectID       string
	firestoreClient *lazy.Loader[*firestore.Client]
	discordSession  *lazy.Loader[*discordgo.Session]
}

func New(ctx context.Context, projectID string) *Clients {
	f := lazy.New(func() (*firestore.Client, error) {
		firestoreClient, err := firestore.NewClient(ctx, projectID)
		if err != nil {
			return nil, fmt.Errorf("failed to create firestore client: %v", err)
		}
		return firestoreClient, nil
	})
	d := lazy.New(func() (*discordgo.Session, error) {
		discordSession, err := discordgo.New("")
		if err != nil {
			return nil, fmt.Errorf("failed to create discord client: %v", err)
		}
		return discordSession, nil
	})
	return &Clients{
		firestoreClient: &f,
		discordSession:  &d,
	}
}

func (c *Clients) Firestore() (*firestore.Client, error) {
	fc := c.firestoreClient.Value()
	if fc == nil {
		return nil, c.firestoreClient.Error()
	}
	return fc, nil
}

func (c *Clients) Discord() (*discordgo.Session, error) {
	return c.discordSession.Value(), c.discordSession.Error()
}
