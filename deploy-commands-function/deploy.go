package main

import (
	"fmt"
	"log"
	"os"

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
				Autocomplete: true,
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
						Autocomplete: true,
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
						Autocomplete: true,
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
						Autocomplete: true,
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
				Autocomplete: true,
			},
		},
	},
	{
		Name:         "stats",
		Description:  "Get stats on the pool",
		Type:         discordgo.ChatApplicationCommand,
		DMPermission: Ptr(false),
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
				Autocomplete: true,
			},
		},
	},
	{
		Name:                     "force-remove",
		Description:              "Force remove a game from the pool. Intended for admins only.",
		Type:                     discordgo.ChatApplicationCommand,
		DefaultMemberPermissions: Ptr(int64(discordgo.PermissionAdministrator)),
		DMPermission:             Ptr(false),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:         "name",
				Description:  "Name of the game/activity",
				Type:         discordgo.ApplicationCommandOptionString,
				Required:     true,
				Autocomplete: true,
			},
		},
	},
}

func main() {
	discordAppID := os.Getenv("DISCORD_APP_ID")
	discordAPIToken := os.Getenv("DISCORD_SECRET_TOKEN")
	if discordAPIToken == "" {
		log.Fatal("Discord token not supplied")
	}
	discordDeploySession, err := discordgo.New(fmt.Sprintf("Bot %v", discordAPIToken))
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}

	_, err = discordDeploySession.ApplicationCommandBulkOverwrite(discordAppID, "", commands)
	if err != nil {
		log.Fatalf("ApplicationCommandBulkOverwrite: %v", err)
	}
}
