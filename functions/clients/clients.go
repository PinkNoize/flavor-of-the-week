package clients

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
)

type lazyLoader[T any] struct {
	t func() (T, error)
}

func newLazyLoader[T any](f func() (T, error)) lazyLoader[T] {
	return lazyLoader[T]{
		t: f,
	}
}

func (l *lazyLoader[T]) value() (T, error) {
	return l.t()
}

type Clients struct {
	Ctx             context.Context
	ProjectID       string
	firestoreClient lazyLoader[*firestore.Client]
}

func New(ctx context.Context, projectID string) Clients {
	f := newLazyLoader(func() (*firestore.Client, error) {
		firestoreClient, err := firestore.NewClient(ctx, projectID)
		if err != nil {
			return nil, fmt.Errorf("failed to create firestore client: %v", err)
		}
		return firestoreClient, nil
	})
	return Clients{
		firestoreClient: f,
	}
}

func (c *Clients) Firestore() (*firestore.Client, error) {
	return c.firestoreClient.value()
}
