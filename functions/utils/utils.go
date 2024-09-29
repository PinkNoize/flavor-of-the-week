package utils

import (
	"encoding/json"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

type Filter struct {
	Name string
	Type string
}

type CustomID struct {
	Type   string
	Filter Filter
	Page   int
}

func NewCustomID(typ string, filter Filter, page int) *CustomID {
	return &CustomID{
		Type:   typ,
		Filter: filter,
		Page:   page,
	}
}

func ParseCustomID(js string) (*CustomID, error) {
	var customID CustomID
	err := json.Unmarshal([]byte(js), &customID)
	if err != nil {
		return nil, fmt.Errorf("Unmarshal: %v", err)
	}
	return &customID, nil
}

func (c *CustomID) ToJson() (string, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

type GameEntry struct {
	Name        string
	Nominations int
}

func BuildDiscordPage(gameEntries []GameEntry, customID *CustomID, isLastPage bool) *discordgo.WebhookEdit {
	embeds := make([]*discordgo.MessageEmbed, 0, len(gameEntries))
	for _, ent := range gameEntries {
		embeds = append(embeds, &discordgo.MessageEmbed{
			Type:        discordgo.EmbedTypeRich,
			Title:       ent.Name,
			Description: fmt.Sprintf("Nominations: %v", ent.Nominations),
		})
	}

	currentPage := customID.Page

	prevPageNum := max(currentPage-1, 0)
	prevCustomID := *customID
	prevCustomID.Page = prevPageNum
	prevCustomIDJson, err := prevCustomID.ToJson()
	if err != nil {
		zap.Error(err)
		prevCustomIDJson = ""
	}

	nextPageNum := currentPage + 1
	nextCustomID := *customID
	nextCustomID.Page = nextPageNum
	nextCustomIDJson, err := nextCustomID.ToJson()
	if err != nil {
		zap.Error(err)
		nextCustomIDJson = ""
	}

	pageTitle := fmt.Sprintf("**Page %v**", currentPage+1)
	return &discordgo.WebhookEdit{
		Content: &pageTitle,
		Embeds:  &embeds,
		Components: &[]discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Prev",
						Style:    discordgo.SecondaryButton,
						Disabled: currentPage == 0,
						CustomID: prevCustomIDJson,
					},
					discordgo.Button{
						Label:    "Next",
						Style:    discordgo.SecondaryButton,
						Disabled: isLastPage,
						CustomID: nextCustomIDJson,
					},
				},
			},
		},
	}
}

func NewWebhookEdit(content string) *discordgo.WebhookEdit {
	return &discordgo.WebhookEdit{
		Content: &content,
	}
}
