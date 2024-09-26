package activity

import (
	"context"
	"crypto/sha256"
	"fmt"
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

// TODO: Maybe change Nominations to an array
type innerActivity struct {
	Typ         ActivityType        `firestore:"type"`
	Name        string              `firestore:"name"`
	Nominations map[string]struct{} `firestore:"nominations"`
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
				FieldPath: []string{"nominations", userId},
				Value:     struct{}{},
			},
		},
	)
	if act.inner.Nominations == nil {
		act.inner.Nominations = map[string]struct{}{userId: {}}
	} else {
		act.inner.Nominations[userId] = struct{}{}
	}
	return err
}

func (act *Activity) RemoveNomination(ctx context.Context, userId string) error {
	_, err := act.docRef.Update(ctx,
		[]firestore.Update{
			{
				FieldPath: []string{"nominations", userId},
				Value:     firestore.Delete,
			},
		},
	)
	delete(act.inner.Nominations, userId)
	return err
}

func GetActivitiesPage(ctx context.Context, guildID, userID, name string, isNominations bool, pageNum int, cl *clients.Clients) ([]utils.GameEntry, bool, error) {
	firestoreClient, err := cl.Firestore()
	if err != nil {
		return nil, false, err
	}
	activityDoc := firestoreClient.Collection(guildID)
	query := activityDoc.Select("name", "nominations")
	if isNominations {
		query = query.WhereEntity(firestore.PropertyPathFilter{
			Path:     firestore.FieldPath{"Nominations", userID},
			Operator: "==",
			Value:    struct{}{},
		})
	}
	if name != "" {
		query = query.WhereEntity(firestore.PropertyFilter{
			Path:     "name",
			Operator: "==",
			Value:    name,
		})
	}
	iter := query.OrderBy("name", firestore.Asc).Offset(pageNum * PAGE_SIZE).Documents(ctx)
	defer iter.Stop()

	results := make([]utils.GameEntry, 0, PAGE_SIZE)
	for {
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
