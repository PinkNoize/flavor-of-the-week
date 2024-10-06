package command

import (
	"context"
	"fmt"

	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
	"github.com/PinkNoize/flavor-of-the-week/functions/utils"
	"github.com/bwmarrin/discordgo"
)

const SEARCH_PAGE_SIZE int = 5

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
	results, totalResults, err := cl.Rawg().SearchGame(ctx, c.Name, c.Page, SEARCH_PAGE_SIZE)
	if err != nil {
		return nil, fmt.Errorf("SearchGame: %v", err)
	}
	entries := make([]utils.GameEntry, 0, len(results))
	for _, res := range results {
		entries = append(entries, utils.GameEntry{
			Name:     res.Name,
			ImageURL: res.ImageBackground,
		})
	}
	isLastPage := totalResults/SEARCH_PAGE_SIZE >= c.Page
	customID := utils.NewCustomID("search",
		utils.Filter{
			Name: c.Name,
		},
		c.Page,
	)
	return utils.BuildDiscordPage(entries, customID, &utils.PageOptions{
		IsLastPage: isLastPage,
		TotalPages: &totalResults,
	}), nil
}
