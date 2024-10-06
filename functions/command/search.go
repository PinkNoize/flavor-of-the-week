package command

import (
	"context"

	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"github.com/PinkNoize/flavor-of-the-week/functions/utils"
	"github.com/bwmarrin/discordgo"
)

type SearchCommand struct {
	Name string
	Page int
}

func NewSearchCommand(name string, page int) *SearchCommand {
	return &SearchCommand{
		Name: name,
		Page: page,
	}
}

func (c *SearchCommand) Execute(ctx context.Context, cl *clients.Clients) (*discordgo.WebhookEdit, error) {
	return utils.NewWebhookEdit("🚧 Command not implemented yet"), nil
}
