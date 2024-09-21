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
		Name:         "add",
		Description:  "Add a game/activity to the pool",
		Type:         discordgo.ChatApplicationCommand,
		DMPermission: Ptr(false),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "type",
				Description: "Type of activities to list",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "Game",
						Value: "game",
					},
					{
						Name:  "Activity",
						Value: "activity",
					},
				},
			},
			{
				Name:         "name",
				Description:  "Name of the game/activity",
				Type:         discordgo.ApplicationCommandOptionString,
				Required:     true,
				Autocomplete: false,
			},
		},
	},
	{
		Name:         "remove",
		Description:  "Remove a game/activity from the pool",
		Type:         discordgo.ChatApplicationCommand,
		DMPermission: Ptr(false),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:         "name",
				Description:  "Name of the game/activity",
				Type:         discordgo.ApplicationCommandOptionString,
				Required:     true,
				Autocomplete: false,
			},
		},
	},
	{
		Name:         "nominations",
		Description:  "Manage nominations for the next poll",
		Type:         discordgo.ChatApplicationCommand,
		DMPermission: Ptr(false),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "add",
				Description: "Nominate a game/activity for the next poll",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:         "name",
						Description:  "Name of the game/activity",
						Type:         discordgo.ApplicationCommandOptionString,
						Required:     true,
						Autocomplete: false,
					},
				},
			},
			{
				Name:        "remove",
				Description: "Remove a nomination for a game/activity",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:         "name",
						Description:  "Name of the game/activity",
						Type:         discordgo.ApplicationCommandOptionString,
						Required:     true,
						Autocomplete: false,
					},
				},
			},
			{
				Name:        "list",
				Description: "List your nominations",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:         "name",
						Description:  "Name of the game/activity",
						Type:         discordgo.ApplicationCommandOptionString,
						Required:     false,
						Autocomplete: false,
					},
				},
			},
		},
	},
	{
		Name:         "pool",
		Description:  "List all games/activities in the pool",
		Type:         discordgo.ChatApplicationCommand,
		DMPermission: Ptr(false),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "type",
				Description: "Type of activities to list",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    false,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "Game",
						Value: "game",
					},
					{
						Name:  "Activity",
						Value: "activity",
					},
				},
			},
			{
				Name:         "name",
				Description:  "Name of the activity to list",
				Type:         discordgo.ApplicationCommandOptionString,
				Required:     false,
				Autocomplete: false,
			},
		},
	},
	// Admin commands
	{
		Name:                     "poll-channel",
		Description:              "Sets the channel to post the poll in",
		Type:                     discordgo.ChatApplicationCommand,
		DefaultMemberPermissions: Ptr(int64(discordgo.PermissionAdministrator)),
		DMPermission:             Ptr(false),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "channel",
				Description: "Channel to send the polls in",
				Type:        discordgo.ApplicationCommandOptionChannel,
				Required:    true,
			},
		},
	},
	{
		Name:                     "start-poll",
		Description:              "Start a poll",
		Type:                     discordgo.ChatApplicationCommand,
		DefaultMemberPermissions: Ptr(int64(discordgo.PermissionAdministrator)),
		DMPermission:             Ptr(false),
	},
	{
		Name:                     "override-fow",
		Description:              "Override the current Flavor of the Week",
		Type:                     discordgo.ChatApplicationCommand,
		DefaultMemberPermissions: Ptr(int64(discordgo.PermissionAdministrator)),
		DMPermission:             Ptr(false),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:         "name",
				Description:  "Name of the game/activity",
				Type:         discordgo.ApplicationCommandOptionString,
				Required:     true,
				Autocomplete: false,
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
