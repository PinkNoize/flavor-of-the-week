package utils

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type GameEntry struct {
	Name        string
	Nominations int
}

func BuildDiscordPage(gameEntries []GameEntry, listType string, currentPage int, isLastPage bool) *discordgo.InteractionResponse {
	embeds := make([]*discordgo.MessageEmbed, 0, len(gameEntries))
	for _, ent := range gameEntries {
		embeds = append(embeds, &discordgo.MessageEmbed{
			Type:        discordgo.EmbedTypeRich,
			Title:       ent.Name,
			Description: fmt.Sprintf("Nominations: %v", ent.Nominations),
		})
	}

	prevPageNum := max(currentPage-1, 0)
	prevPageLabel := ""
	if currentPage != 0 {
		prevPageLabel = fmt.Sprintf("%v", prevPageNum)
	}

	nextPageNum := currentPage + 1
	nextPageLabel := ""
	if !isLastPage {
		nextPageLabel = fmt.Sprintf("%v", nextPageNum)
	}

	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("**Page %v**", currentPage),
			Embeds:  embeds,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    prevPageLabel,
							Style:    discordgo.SecondaryButton,
							Disabled: currentPage == 0,
							CustomID: fmt.Sprintf("%v: %v", listType, prevPageNum),
						},
						discordgo.Button{
							Label:    nextPageLabel,
							Style:    discordgo.SecondaryButton,
							Disabled: isLastPage,
							CustomID: fmt.Sprintf("%v: %v", listType, nextPageNum),
						},
					},
				},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}
}
