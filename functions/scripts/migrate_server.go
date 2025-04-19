package main

import (
	"context"
	"fmt"
	"os"

	"cloud.google.com/go/firestore"
	"github.com/PinkNoize/flavor-of-the-week/functions/activity"
	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"go.uber.org/zap"
	"google.golang.org/api/iterator"
)

func main() {
	args := os.Args
	if len(args) != 3 {
		fmt.Printf("%s <src server> <dest server>", args[0])
		return
	}
	logger := zap.L()
	ctx := context.Background()
	source_server := args[1]
	dest_server := args[2]
	project := os.Getenv("GOOGLE_CLOUD_PROJECT")
	client := clients.New(ctx, project, "", "")

	fs, err := client.Firestore()
	if err != nil {
		logger.Fatal(fmt.Sprintf("firestore: %s", err))
		return
	}
	fow_collection := fs.Collection("flavor-of-the-week-main")
	// Get all activities in src server
	query := fow_collection.WhereEntity(firestore.PropertyFilter{
		Path:     "guild_id",
		Operator: "==",
		Value:    source_server,
	})

	docCounter := 0
	iter := query.Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		var inAct activity.InnerActivity
		err = doc.DataTo(&inAct)
		if err != nil {
			logger.Sugar().Fatalf("doc.DataTo: %v", err)
		}

		// Copy to new server
		_, err = activity.Create(ctx, inAct.Typ, inAct.Name, dest_server, inAct.GameInfo, client)
		if err != nil {
			logger.Sugar().Fatalf("activity.Create: %s", err)
		}

		act, err := activity.GetActivity(ctx, inAct.Name, source_server, client)
		if err != nil {
			logger.Sugar().Fatalf("activity.GetActivity: %s", err)
		}

		err = act.RemoveActivity(ctx, true)
		if err != nil {
			logger.Sugar().Errorf("Failed to remove: %s", inAct.Name)
		}
		docCounter += 1
		if docCounter%10 == 0 {
			fmt.Printf("Migrated %v activities\n", docCounter)
		}
	}
	fmt.Printf("Copied %v activites from %v to %v", docCounter, source_server, dest_server)
}
