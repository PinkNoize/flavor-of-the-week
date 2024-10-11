package command

import (
	"context"

	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"github.com/bwmarrin/discordgo"
)

type HelpCommand struct {
}

func NewHelpCommand() *HelpCommand {
	return &HelpCommand{}
}

func (c *HelpCommand) Execute(ctx context.Context, cl *clients.Clients) (*discordgo.WebhookEdit, error) {
	return &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{
			{
				Title: "Flavor of the Week",
				Description: `Flavor of the Week is a bot to maintain a backlog of games/activities and to automate voting on the "flavor of the week".

You can view all commands by clicking on the apps button in your message bar.
Here are some of the important commands:`,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name: "The pool",
						Value: "The pool holds all games and activites. You can view the pool with `/pool`\n" +
							"You can remove an item with `/remove`",
					},
					{
						Name: "Adding a game or activity to the pool",
						Value: "You can add a game by\n" +
							" 1. Adding it with `/add`\n" +
							" 2. Searching for a game with `/search` and then selecting the game from the results",
					},
					{
						Name: "Nominations",
						Value: "You can suggest games to be added to the next poll by nominating them. All nominations are cleared after a poll.\n" +
							"All nomination commands can be found in `/nominations`",
					},
				},
			},
		},
	}, nil
}
