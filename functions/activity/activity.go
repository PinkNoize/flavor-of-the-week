package activity

import (
	"context"
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"github.com/PinkNoize/flavor-of-the-week/functions/utils"
	"github.com/bwmarrin/discordgo"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const PAGE_SIZE int = 5
const MAX_AUTOCOMPLETE_ENTRIES int = 5

type ActivityType string

const (
	GAME     = "GAME"
	ACTIVITY = "ACTIVITY"
)

type ActivityErrorReason string

const (
	ALREADY_EXISTS        = "already exists"
	DOES_NOT_EXIST        = "does not exist"
	STILL_HAS_NOMINATIONS = "still has nominations"
)

type ActivityError struct {
	Reason ActivityErrorReason
}

func NewActivityError(r ActivityErrorReason) *ActivityError {
	return &ActivityError{Reason: r}
}

func (actErr *ActivityError) Error() string {
	return string(actErr.Reason)
}

type innerActivity struct {
	Typ         ActivityType `firestore:"type"`
	Name        string       `firestore:"name"`
	SearchName  string       `firestore:"search_name"`
	GuildID     string       `firestore:"guild_id"`
	Nominations []string     `firestore:"nominations"`
	IsFow       bool         `firestore:"is_fow"`
}

type Activity struct {
	docName    string
	inner      innerActivity
	docRef     *firestore.DocumentRef
	updateTime time.Time
}

func getCollection(cl *clients.Clients) (*firestore.CollectionRef, error) {
	firestoreClient, err := cl.Firestore()
	if err != nil {
		return nil, err
	}
	return firestoreClient.Collection("flavor-of-the-week"), nil
}

func GetActivity(ctx context.Context, name, guildID string, cl *clients.Clients) (*Activity, error) {
	activityCollection, err := getCollection(cl)
	if err != nil {
		return nil, fmt.Errorf("getCollection: %v", err)
	}
	docName := generateName(guildID, name)
	activityDoc := activityCollection.Doc(docName)
	activityDocSnap, err := activityDoc.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, NewActivityError(DOES_NOT_EXIST)
		}
		return nil, err
	}
	act := Activity{
		docName:    docName,
		docRef:     activityDoc,
		updateTime: activityDocSnap.UpdateTime,
	}
	err = activityDocSnap.DataTo(&act.inner)
	if err != nil {
		return nil, fmt.Errorf("Failed to deserialize activity: %v", err)
	}
	return &act, nil
}

func generateName(guildId, name string) string {
	return fmt.Sprintf("%v:%x", guildId, sha256.Sum256([]byte(name)))
}

func Create(ctx context.Context, typ ActivityType, name, guildID string, cl *clients.Clients) (*Activity, error) {
	activityCollection, err := getCollection(cl)
	if err != nil {
		return nil, fmt.Errorf("getCollection: %v", err)
	}

	docName := generateName(guildID, name)
	activityDoc := activityCollection.Doc(docName)
	inAct := innerActivity{
		Typ:        typ,
		Name:       name,
		SearchName: strings.ToLower(name),
		GuildID:    guildID,
	}
	ctxzap.Info(ctx, fmt.Sprintf("Creating %v in %v", name, guildID))
	wr, err := activityDoc.Create(ctx, &inAct)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return nil, NewActivityError(ALREADY_EXISTS)
		}
		return nil, fmt.Errorf("activityDoc.Create: %v", err)
	}
	return &Activity{
		docName:    docName,
		inner:      inAct,
		docRef:     activityDoc,
		updateTime: wr.UpdateTime,
	}, nil
}

func (act *Activity) RemoveActivity(ctx context.Context, force bool) error {
	if force {
		_, err := act.docRef.Delete(ctx)
		if err != nil {
			return fmt.Errorf("Failed to delete %v (%v)", act.docName, act.inner.Name)
		}
		return nil
	}
	if len(act.inner.Nominations) > 0 {
		return NewActivityError(STILL_HAS_NOMINATIONS)
	}
	_, err := act.docRef.Delete(ctx, firestore.LastUpdateTime(act.updateTime))
	if err != nil {
		return fmt.Errorf("Failed to delete %v (%v)", act.docName, act.inner.Name)
	}
	return nil
}

func (act *Activity) AddNomination(ctx context.Context, userId string) error {
	_, err := act.docRef.Update(ctx,
		[]firestore.Update{
			{
				FieldPath: firestore.FieldPath{"nominations"},
				Value:     firestore.ArrayUnion(userId),
			},
		},
	)
	if !slices.Contains(act.inner.Nominations, userId) {
		act.inner.Nominations = append(act.inner.Nominations, userId)
	}
	return err
}

func (act *Activity) RemoveNomination(ctx context.Context, userId string) error {
	_, err := act.docRef.Update(ctx,
		[]firestore.Update{
			{
				FieldPath: firestore.FieldPath{"nominations"},
				Value:     firestore.ArrayRemove(userId),
			},
		},
	)
	act.inner.Nominations = slices.DeleteFunc(act.inner.Nominations, func(cmp string) bool {
		return cmp == userId
	})
	return err
}

type ActivitesPageOptions struct {
	Name            string
	Type            ActivityType
	NominationsOnly bool
	UserId          string
}

func GetActivitiesPage(ctx context.Context, guildID string, pageNum int, opts *ActivitesPageOptions, cl *clients.Clients) ([]utils.GameEntry, bool, error) {
	// Shortcut to get entries if name is specified
	if opts.Name != "" {
		act, err := GetActivity(ctx, opts.Name, guildID, cl)
		if err != nil {
			ae, ok := err.(*ActivityError)
			if ok && ae.Reason == DOES_NOT_EXIST {
				return []utils.GameEntry{}, true, nil
			}
			return nil, false, fmt.Errorf("GetActivity: %v", err)
		}
		if !opts.NominationsOnly || slices.Contains(act.inner.Nominations, opts.UserId) {
			return []utils.GameEntry{
				{
					Name:        opts.Name,
					Nominations: len(act.inner.Nominations),
				},
			}, true, nil
		}
		return []utils.GameEntry{}, true, nil
	}

	activityCollection, err := getCollection(cl)
	if err != nil {
		return nil, false, fmt.Errorf("getCollection: %v", err)
	}
	// This query requires an index which is created in terraform
	query := activityCollection.Select("name", "nominations").WhereEntity(firestore.PropertyFilter{
		Path:     "guild_id",
		Operator: "==",
		Value:    guildID,
	})
	if opts.Type != "" {
		query = query.WhereEntity(firestore.PropertyFilter{
			Path:     "type",
			Operator: "==",
			Value:    opts.Type,
		})
	}
	if opts.NominationsOnly {
		query = query.WhereEntity(firestore.PropertyFilter{
			Path:     "nominations",
			Operator: "array-contains",
			Value:    opts.UserId,
		})
	}
	iter := query.OrderBy("name", firestore.Asc).Offset(pageNum * PAGE_SIZE).Documents(ctx)
	defer iter.Stop()

	results := make([]utils.GameEntry, 0, PAGE_SIZE)
	for i := 0; i < PAGE_SIZE; i++ {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, false, fmt.Errorf("iter.Next: %v", err)
		}
		var inAct innerActivity
		err = doc.DataTo(&inAct)
		if err != nil {
			return nil, false, fmt.Errorf("doc.DataTo: %v", err)
		}
		results = append(results, utils.GameEntry{
			Name:        inAct.Name,
			Nominations: len(inAct.Nominations),
		})
	}
	lastItem := false
	_, err = iter.Next()
	if err != nil {
		if err == iterator.Done {
			lastItem = true
		}
	}
	return results, lastItem, nil

}

func AutocompleteActivities(ctx context.Context, guildID, text string, cl *clients.Clients) ([]*discordgo.ApplicationCommandOptionChoice, error) {
	activityCollection, err := getCollection(cl)
	if err != nil {
		return []*discordgo.ApplicationCommandOptionChoice{}, fmt.Errorf("getCollection: %v", err)
	}
	// This query requires an index which is created in terraform
	query := activityCollection.Select("name").WhereEntity(firestore.PropertyFilter{
		Path:     "guild_id",
		Operator: "==",
		Value:    guildID,
	}).WhereEntity(firestore.PropertyFilter{
		Path:     "search_name",
		Operator: ">=",
		Value:    strings.ToLower(text),
	}).WhereEntity(firestore.PropertyFilter{
		Path:     "search_name",
		Operator: "<=",
		Value:    "\uf8ff",
	}).OrderBy("name", firestore.Asc)
	iter := query.Documents(ctx)
	defer iter.Stop()

	results := make([]*discordgo.ApplicationCommandOptionChoice, 0, MAX_AUTOCOMPLETE_ENTRIES)
	for i := 0; i < MAX_AUTOCOMPLETE_ENTRIES; i++ {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return []*discordgo.ApplicationCommandOptionChoice{}, fmt.Errorf("iter.Next: %v", err)
		}
		var inAct innerActivity
		err = doc.DataTo(&inAct)
		if err != nil {
			return []*discordgo.ApplicationCommandOptionChoice{}, fmt.Errorf("doc.DataTo: %v", err)
		}
		results = append(results, &discordgo.ApplicationCommandOptionChoice{
			Name:  inAct.Name,
			Value: inAct.Name,
		})
	}
	return results, nil
}
