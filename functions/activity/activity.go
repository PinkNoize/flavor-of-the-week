package activity

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math/rand"
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

type randomHelper struct {
	Num1 uint32 `firestore:"num_1"`
	Num2 uint32 `firestore:"num_2"`
}

func NewRandomHelper() randomHelper {
	return randomHelper{
		Num1: rand.Uint32(),
		Num2: rand.Uint32(),
	}
}

type GameInfo struct {
	Id              int    `firestore:"id"`
	Slug            string `firestore:"slug"`
	BackgroundImage string `firestore:"bg_image"`
}

type innerActivity struct {
	Typ              ActivityType `firestore:"type"`
	Name             string       `firestore:"name"`
	SearchName       string       `firestore:"search_name"`
	GuildID          string       `firestore:"guild_id"`
	Nominations      []string     `firestore:"nominations"`
	NominationsCount int          `firestore:"nominations_count"`
	Random           randomHelper `firestore:"random"`
	GameInfo         *GameInfo    `firestore:"game_info"`
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

func Create(ctx context.Context, typ ActivityType, name, guildID string, gameInfo *GameInfo, cl *clients.Clients) (*Activity, error) {
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
		Random:     NewRandomHelper(),
		GameInfo:   gameInfo,
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
	if slices.Contains(act.inner.Nominations, userId) {
		return nil
	}
	_, err := act.docRef.Update(ctx,
		[]firestore.Update{
			{
				FieldPath: firestore.FieldPath{"nominations"},
				Value:     firestore.ArrayUnion(userId),
			},
			{
				FieldPath: firestore.FieldPath{"nominations_count"},
				Value:     firestore.Increment(1),
			},
			{
				FieldPath: firestore.FieldPath{"random"},
				Value:     NewRandomHelper(),
			},
		},
	)
	act.inner.Nominations = append(act.inner.Nominations, userId)
	act.inner.NominationsCount += 1
	return err
}

func (act *Activity) RemoveNomination(ctx context.Context, userId string) error {
	if !slices.Contains(act.inner.Nominations, userId) {
		return nil
	}
	_, err := act.docRef.Update(ctx,
		[]firestore.Update{
			{
				FieldPath: firestore.FieldPath{"nominations"},
				Value:     firestore.ArrayRemove(userId),
			},
			{
				FieldPath: firestore.FieldPath{"nominations_count"},
				Value:     firestore.Increment(-1),
			},
			{
				FieldPath: firestore.FieldPath{"random"},
				Value:     NewRandomHelper(),
			},
		},
	)
	act.inner.Nominations = slices.DeleteFunc(act.inner.Nominations, func(cmp string) bool {
		return cmp == userId
	})
	act.inner.NominationsCount -= 1
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
		imageUrl := ""
		if act.inner.GameInfo != nil {
			imageUrl = act.inner.GameInfo.BackgroundImage
		}
		if !opts.NominationsOnly || slices.Contains(act.inner.Nominations, opts.UserId) {
			return []utils.GameEntry{
				{
					Name:        opts.Name,
					Nominations: firestore.Ptr(len(act.inner.Nominations)),
					ImageURL:    imageUrl,
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
	query := activityCollection.Select("name", "nominations", "game_info")
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
	query = query.WhereEntity(firestore.PropertyFilter{
		Path:     "guild_id",
		Operator: "==",
		Value:    guildID,
	})
	iter := query.OrderBy("search_name", firestore.Asc).Offset(pageNum * PAGE_SIZE).Documents(ctx)
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
		imageUrl := ""
		if inAct.GameInfo != nil {
			imageUrl = inAct.GameInfo.BackgroundImage
		}
		results = append(results, utils.GameEntry{
			Name:        inAct.Name,
			Nominations: firestore.Ptr(len(inAct.Nominations)),
			ImageURL:    imageUrl,
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
	text = strings.ToLower(text)
	// This query requires an index which is created in terraform
	query := activityCollection.Select("name").WhereEntity(firestore.PropertyFilter{
		Path:     "search_name",
		Operator: ">=",
		Value:    text,
	}).WhereEntity(firestore.PropertyFilter{
		Path:     "search_name",
		Operator: "<=",
		Value:    text + "\uf8ff",
	}).WhereEntity(firestore.PropertyFilter{
		Path:     "guild_id",
		Operator: "==",
		Value:    guildID,
	}).OrderBy("search_name", firestore.Asc)
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

func AutocompleteGames(ctx context.Context, guildID, text string, cl *clients.Clients) ([]*discordgo.ApplicationCommandOptionChoice, error) {
	gamesList, count, err := cl.Rawg().SearchGame(ctx, text, 1, MAX_AUTOCOMPLETE_ENTRIES)
	if err != nil {
		return []*discordgo.ApplicationCommandOptionChoice{}, fmt.Errorf("Rawg.SearchGame: %v", err)
	}

	results := make([]*discordgo.ApplicationCommandOptionChoice, 0, count)
	for _, game := range gamesList {
		results = append(results, &discordgo.ApplicationCommandOptionChoice{
			Name:  game.Name,
			Value: game.Slug,
		})
	}

	return results, nil
}

func GetTopNominations(ctx context.Context, guildID string, n int, cl *clients.Clients) ([]string, error) {
	activityCollection, err := getCollection(cl)
	if err != nil {
		return nil, fmt.Errorf("getCollection: %v", err)
	}
	// This query requires an index which is created in terraform
	query := activityCollection.Select("name").WhereEntity(&firestore.PropertyFilter{
		Path:     "nominations_count",
		Operator: ">",
		Value:    0,
	}).WhereEntity(&firestore.PropertyFilter{
		Path:     "guild_id",
		Operator: "==",
		Value:    guildID,
	}).OrderBy("nominations_count", firestore.Desc).Limit(n)
	iter := query.Documents(ctx)
	defer iter.Stop()

	results := make([]string, 0, n)
	for i := 0; i < n; i++ {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("iter.Next: %v", err)
		}
		var inAct innerActivity
		err = doc.DataTo(&inAct)
		if err != nil {
			return nil, fmt.Errorf("doc.DataTo: %v", err)
		}
		results = append(results, inAct.Name)
	}
	return results, nil
}

func GetRandomActivities(ctx context.Context, guildID string, n int, cl *clients.Clients) ([]string, error) {
	activityCollection, err := getCollection(cl)
	if err != nil {
		return nil, fmt.Errorf("getCollection: %v", err)
	}
	randomNumber := rand.Uint32()
	randomSelector := (rand.Int() % 2) + 1
	randomPath := fmt.Sprintf("random.num_%v", randomSelector)
	// This query requires an index which is created in terraform
	query := activityCollection.Select("name").WhereEntity(&firestore.PropertyFilter{
		Path:     randomPath,
		Operator: ">=",
		Value:    randomNumber,
	}).WhereEntity(&firestore.PropertyFilter{
		Path:     "guild_id",
		Operator: "==",
		Value:    guildID,
	}).OrderBy(randomPath, firestore.Asc).Limit(n)
	iter := query.Documents(ctx)
	defer iter.Stop()

	results := make([]string, 0, n)
	for i := 0; i < n; i++ {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("iter.Next: %v", err)
		}
		var inAct innerActivity
		err = doc.DataTo(&inAct)
		if err != nil {
			return nil, fmt.Errorf("doc.DataTo: %v", err)
		}
		results = append(results, inAct.Name)
	}
	return results, nil
}

func ClearNominations(ctx context.Context, guildID string, cl *clients.Clients) error {
	firestoreClient, err := cl.Firestore()
	if err != nil {
		return fmt.Errorf("Firestore: %v", err)
	}
	activityCollection, err := getCollection(cl)
	if err != nil {
		return fmt.Errorf("getCollection: %v", err)
	}
	// This query requires an index which is created in terraform
	// Sorted in descending to use the same index as top nominations
	query := activityCollection.Select().WhereEntity(&firestore.PropertyFilter{
		Path:     "nominations_count",
		Operator: ">",
		Value:    0,
	}).WhereEntity(&firestore.PropertyFilter{
		Path:     "guild_id",
		Operator: "==",
		Value:    guildID,
	}).OrderBy("nominations_count", firestore.Desc)
	iter := query.Documents(ctx)
	bulkWriter := firestoreClient.BulkWriter(ctx)
	defer bulkWriter.End()

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("iter.Next: %v", err)
		}
		_, err = bulkWriter.Update(doc.Ref, []firestore.Update{
			{
				Path:  "nominations",
				Value: []string{},
			},
			{
				Path:  "nominations_count",
				Value: 0,
			},
			{
				FieldPath: firestore.FieldPath{"random"},
				Value:     NewRandomHelper(),
			},
		})
		if err != nil {
			return fmt.Errorf("Update: %v", err)
		}
	}
	return nil
}
