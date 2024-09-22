package clients

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/josestg/lazy"
)

type Clients struct {
	Ctx             context.Context
	ProjectID       string
	firestoreClient *lazy.Loader[*firestore.Client]
}

func New(ctx context.Context, projectID string) *Clients {
	f := lazy.New(func() (*firestore.Client, error) {
		firestoreClient, err := firestore.NewClient(ctx, projectID)
		if err != nil {
			return nil, fmt.Errorf("failed to create firestore client: %v", err)
		}
		return firestoreClient, nil
	})
	return &Clients{
		firestoreClient: &f,
	}
}

func (c *Clients) Firestore() (*firestore.Client, error) {
	return c.firestoreClient.Value(), c.firestoreClient.Error()
}
