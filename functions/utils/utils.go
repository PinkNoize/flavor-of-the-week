package utils

import (
	"fmt"

	"github.com/PinkNoize/flavor-of-the-week/functions/customid"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

type PageOptions struct {
	TotalPages *int
	IsLastPage bool
}

type GameEntry struct {
	Name        string
	Nominations *int
	ImageURL    string
}

// This needs to be refactored with some kind of options factory
func BuildDiscordPage(gameEntries []GameEntry, customID *customid.CustomID, pageOpt *PageOptions, selectMenu *discordgo.SelectMenu) *discordgo.WebhookEdit {
	embeds := make([]*discordgo.MessageEmbed, 0, len(gameEntries))
	for _, ent := range gameEntries {
		var thumbnail *discordgo.MessageEmbedThumbnail
		if ent.ImageURL != "" {
			thumbnail = &discordgo.MessageEmbedThumbnail{
				URL: ent.ImageURL,
			}
		}
		description := ""
		if ent.Nominations != nil {
			description = fmt.Sprintf("Nominations: %v", *ent.Nominations)
		}
		embeds = append(embeds, &discordgo.MessageEmbed{
			Type:        discordgo.EmbedTypeRich,
			Title:       ent.Name,
			Description: description,
			Thumbnail:   thumbnail,
		})
	}

	currentPage := customID.Page

	prevPageNum := max(currentPage-1, 0)
	prevCustomID := *customID
	prevCustomID.Page = prevPageNum
	prevCustomIDJson, err := prevCustomID.ToDiscordCustomID()
	if err != nil {
		zap.Error(err)
		prevCustomIDJson = ""
	}

	nextPageNum := currentPage + 1
	nextCustomID := *customID
	nextCustomID.Page = nextPageNum
	nextCustomIDJson, err := nextCustomID.ToDiscordCustomID()
	if err != nil {
		zap.Error(err)
		nextCustomIDJson = ""
	}

	var pageLabel string
	if pageOpt.TotalPages != nil {
		if currentPage >= *pageOpt.TotalPages-1 {
			pageOpt.IsLastPage = true
		} else {
			pageOpt.IsLastPage = false
		}
		pageLabel = fmt.Sprintf("%v/%v", currentPage+1, max(*pageOpt.TotalPages, 1))
	} else {
		pageLabel = fmt.Sprintf("%v/??", currentPage+1)
	}

	components := make([]discordgo.MessageComponent, 0, 2)
	if selectMenu != nil {
		components = append(components, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				selectMenu,
			},
		})
	}
	components = append(components, discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Prev",
				Style:    discordgo.SecondaryButton,
				Disabled: currentPage == 0,
				CustomID: prevCustomIDJson,
			},
			discordgo.Button{
				Label:    pageLabel,
				Style:    discordgo.SecondaryButton,
				Disabled: true,
				CustomID: "{}",
			},
			discordgo.Button{
				Label:    "Next",
				Style:    discordgo.SecondaryButton,
				Disabled: pageOpt.IsLastPage,
				CustomID: nextCustomIDJson,
			},
		},
	})

	return &discordgo.WebhookEdit{
		Embeds:     &embeds,
		Components: &components,
	}
}

func NewWebhookEdit(content string) *discordgo.WebhookEdit {
	return &discordgo.WebhookEdit{
		Content: &content,
	}
}

func OptionsToMap(opts []*discordgo.ApplicationCommandInteractionDataOption) map[string]*discordgo.ApplicationCommandInteractionDataOption {
	mappedOpts := make(map[string]*discordgo.ApplicationCommandInteractionDataOption)

	for i := range opts {
		if opts[i] != nil {
			mappedOpts[opts[i].Name] = opts[i]
		}
	}
	return mappedOpts
}

func VerifyOpts(opts map[string]*discordgo.ApplicationCommandInteractionDataOption, expected []string) (bool, string) {
	for _, v := range expected {
		if _, ok := opts[v]; !ok {
			return false, v
		}
	}
	return true, ""
}
