package main

import (
	"context"
	"fmt"
	"log"
	"os"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/bwmarrin/discordgo"
)

func Ptr[T any](v T) *T {
	return &v
}

var commands = []*discordgo.ApplicationCommand{
	// User commands
	{
		Name:         "fow",
		Description:  "Flavor of the Week",
		DMPermission: Ptr(false),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "set-poll-channel",
				Description: "Sets the poll channel",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:        "channel",
						Description: "Channel to send the polls",
						Type:        discordgo.ApplicationCommandOptionChannel,
					},
				},
			},
		},
	},
	// Admin commands
	{
		Name:                     "fow-setup",
		Description:              "Flavor of the Week",
		DefaultMemberPermissions: Ptr(int64(discordgo.PermissionAdministrator)),
		DMPermission:             Ptr(false),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "set-poll-channel",
				Description: "Sets the poll channel",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:        "channel",
						Description: "Channel to send the polls",
						Type:        discordgo.ApplicationCommandOptionChannel,
					},
				},
			},
		},
	},
}

func main() {
	discordSecretID := os.Getenv("DISCORD_SECRET_ID")
	discordAppID := os.Getenv("DISCORD_APP_ID")
	ctx := context.Background()

	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		log.Fatalf("Error: NewClient: %v", err)
	}
	defer client.Close()
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("%v/versions/latest", discordSecretID),
	}
	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		log.Fatalf("Error: AccessSecretVersion: %v", err)
	}
	discordAPIToken := string(result.Payload.Data)
	discordDeploySession, err := discordgo.New(fmt.Sprintf("Bot %v", discordAPIToken))
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}

	for i := range commands {
		_, err := discordDeploySession.ApplicationCommandCreate(discordAppID, "", commands[i])
		if err != nil {
			log.Fatalf("ApplicationCommandCreate: %v", err)
		}
	}
}
