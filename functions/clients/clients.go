package clients

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	"github.com/bwmarrin/discordgo"
	"github.com/josestg/lazy"
	"google.golang.org/api/googleapi"
)

type Clients struct {
	Ctx             context.Context
	ProjectID       string
	firestoreClient *lazy.Loader[*firestore.Client]
	discordSession  *lazy.Loader[*discordgo.Session]
	rawgClient      *lazy.Loader[*Rawg]
	bannedUsers     *lazy.Loader[map[string]struct{}]
}

func New(ctx context.Context, projectID, discordToken, rawgToken string) *Clients {
	f := lazy.New(func() (*firestore.Client, error) {
		firestoreClient, err := firestore.NewClient(ctx, projectID)
		if err != nil {
			return nil, fmt.Errorf("failed to create firestore client: %v", err)
		}
		return firestoreClient, nil
	})
	d := lazy.New(func() (*discordgo.Session, error) {
		discordSession, err := discordgo.New("Bot " + discordToken)
		if err != nil {
			return nil, fmt.Errorf("failed to create discord client: %v", err)
		}
		return discordSession, nil
	})
	r := lazy.New(func() (*Rawg, error) {
		return NewRawg(rawgToken), nil
	})
	bU := lazy.New(func() (map[string]struct{}, error) {
		client, err := storage.NewClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create storage client: %v", err)
		}
		rc, err := client.Bucket(os.Getenv("RESOURCES_BUCKET")).Object("banned-users.json").NewReader(ctx)
		if err != nil {
			// return empty list if doesn't exist
			var e *googleapi.Error
			if ok := errors.As(err, &e); ok {
				if e.Code == 404 {
					return nil, nil
				}
			}
			return nil, fmt.Errorf("failed to read object: %v", err)
		}
		defer rc.Close()
		body, err := io.ReadAll(rc)
		if err != nil {
			return nil, fmt.Errorf("readAll: %v", err)
		}
		var userList []string
		err = json.Unmarshal(body, &userList)
		if err != nil {
			return nil, fmt.Errorf("Unmarshal: %v", err)
		}
		userLookup := make(map[string]struct{})
		for _, user := range userList {
			userLookup[user] = struct{}{}
		}
		return userLookup, nil
	})
	return &Clients{
		firestoreClient: &f,
		discordSession:  &d,
		rawgClient:      &r,
		bannedUsers:     &bU,
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

func (c *Clients) Rawg() *Rawg {
	return c.rawgClient.Value()
}

func (c *Clients) BannedUsers() (map[string]struct{}, error) {
	return c.bannedUsers.Value(), c.bannedUsers.Error()
}
