package activity

import (
	"context"
	"crypto/sha256"
	"fmt"
	"slices"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"github.com/PinkNoize/flavor-of-the-week/functions/utils"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const PAGE_SIZE int = 5

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
	Nominations []string     `firestore:"nominations"`
}

type Activity struct {
	docName    string
	inner      innerActivity
	docRef     *firestore.DocumentRef
	updateTime time.Time
}

func GetActivity(ctx context.Context, name, guildID string, cl *clients.Clients) (*Activity, error) {
	firestoreClient, err := cl.Firestore()
	if err != nil {
		return nil, err
	}
	docName := generateName(name)
	activityDoc := firestoreClient.Collection(guildID).Doc(docName)
	activityDocSnap, err := activityDoc.Get(ctx)
	if err != nil {
		return nil, err
	}
	if !activityDocSnap.Exists() {
		return nil, NewActivityError(DOES_NOT_EXIST)
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

func generateName(name string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(name)))
}

func Create(ctx context.Context, typ ActivityType, name, guildID string, cl *clients.Clients) (*Activity, error) {
	firestoreClient, err := cl.Firestore()
	if err != nil {
		return nil, err
	}
	docName := generateName(name)
	activityDoc := firestoreClient.Collection(guildID).Doc(docName)
	inAct := innerActivity{
		Typ:  typ,
		Name: name,
	}
	ctxzap.Info(ctx, "Creating ")
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

func (act *Activity) RemoveActivity(ctx context.Context) error {
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

func GetActivitiesPage(ctx context.Context, guildID, userID, name string, searchNominations bool, pageNum int, cl *clients.Clients) ([]utils.GameEntry, bool, error) {
	// Shortcut to get entries if name is specified
	if name != "" {
		act, err := GetActivity(ctx, name, guildID, cl)
		if err != nil {
			ae, ok := err.(*ActivityError)
			if ok && ae.Reason == DOES_NOT_EXIST {
				return []utils.GameEntry{}, true, nil
			}
			return nil, false, fmt.Errorf("GetActivity: %v", err)
		}
		if !searchNominations || slices.Contains(act.inner.Nominations, userID) {
			return []utils.GameEntry{
				{
					Name:        name,
					Nominations: len(act.inner.Nominations),
				},
			}, true, nil
		}
		return []utils.GameEntry{}, true, nil
	}

	firestoreClient, err := cl.Firestore()
	if err != nil {
		return nil, false, err
	}
	activityDoc := firestoreClient.Collection(guildID)
	// This query requires an index which is created in terraform??
	query := activityDoc.Select("name", "nominations")
	if searchNominations {
		query = query.WhereEntity(firestore.PropertyFilter{
			Path:     "nominations",
			Operator: "array-contains",
			Value:    userID,
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
