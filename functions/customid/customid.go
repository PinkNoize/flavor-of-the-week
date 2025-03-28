package customid

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"github.com/PinkNoize/flavor-of-the-week/functions/setup"
	"github.com/google/uuid"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
)

const TTL time.Duration = time.Hour * 24

type Filter struct {
	Name string
	Type string
}

type innerCustomID struct {
	Timestamp time.Time `firestore:"timestamp"`
	Type      string    `firestore:"type"`
	Filter    Filter    `firestore:"filter"`
}

type CustomID struct {
	innerCustomID innerCustomID
	Page          int
	docName       string
	docRef        *firestore.DocumentRef
}

type outCustomID struct {
	Type string  `json:"type"`
	ID   *string `json:"id"`
	Page int     `json:"page"`
}

func getCollection(cl *clients.Clients) (*firestore.CollectionRef, error) {
	firestoreClient, err := cl.Firestore()
	if err != nil {
		return nil, err
	}
	return firestoreClient.Collection(fmt.Sprintf("flavor-of-the-week-state-%v", setup.ENV)), nil
}

func CreateCustomID(ctx context.Context, typ string, filter Filter, page int, cl *clients.Clients) (*CustomID, error) {
	stateCollection, err := getCollection(cl)
	if err != nil {
		return nil, fmt.Errorf("getCollection: %v", err)
	}
	docName := uuid.New().String()
	stateDoc := stateCollection.Doc(docName)
	inCID := innerCustomID{
		Timestamp: time.Now().Add(TTL),
		Type:      typ,
		Filter:    filter,
	}
	ctxzap.Info(ctx, fmt.Sprintf("Creating %v in state collection", docName))
	_, err = stateDoc.Create(ctx, &inCID)
	if err != nil {
		return nil, fmt.Errorf("stateDoc.Create: %v", err)
	}
	return &CustomID{
		innerCustomID: inCID,
		Page:          page,
		docName:       docName,
		docRef:        stateDoc,
	}, nil
}

func GetCustomID(ctx context.Context, discordCustomID string, cl *clients.Clients) (*CustomID, error) {
	var oCustomID outCustomID
	err := json.Unmarshal([]byte(discordCustomID), &oCustomID)
	if err != nil {
		return nil, fmt.Errorf("unmarshal: %v", err)
	}
	if oCustomID.ID == nil {
		return &CustomID{
			innerCustomID: innerCustomID{
				Type: oCustomID.Type,
			},
		}, nil
	}

	stateCollection, err := getCollection(cl)
	if err != nil {
		return nil, fmt.Errorf("getCollection: %v", err)
	}
	stateDoc := stateCollection.Doc(*oCustomID.ID)
	docSnap, err := stateDoc.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("get: %v", err)
	}
	id := CustomID{
		docName: *oCustomID.ID,
		docRef:  stateDoc,
		Page:    oCustomID.Page,
	}
	err = docSnap.DataTo(&id.innerCustomID)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize customID: %v", err)
	}

	return &id, nil
}

func (c *CustomID) ToDiscordCustomID() (string, error) {
	oCustomID := outCustomID{
		Type: c.innerCustomID.Type,
		ID:   &c.docName,
		Page: c.Page,
	}

	b, err := json.Marshal(oCustomID)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (c *CustomID) Type() string {
	return c.innerCustomID.Type
}

func (c *CustomID) Filter() Filter {
	return c.innerCustomID.Filter
}
