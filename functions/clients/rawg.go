package clients

import (
	"context"
	"fmt"
	"net/http"

	"github.com/dimuska139/rawg-sdk-go/v3"
)

type Rawg struct {
	client *rawg.Client
}

func NewRawg(rawgToken string) *Rawg {
	return &Rawg{
		rawg.NewClient(http.DefaultClient, &rawg.Config{
			ApiKey:   rawgToken,
			Language: "us",
			Rps:      5,
		}),
	}
}

func (r *Rawg) GetGame(ctx context.Context, game string) (*rawg.GameDetailed, error) {
	detail, err := r.client.GetGame(ctx, game)
	return detail, fmt.Errorf("rawg.go: %v", err)
}
