package clients

import (
	"context"
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
	return r.client.GetGame(ctx, game)
}

func RawgErrorMsg(err error) string {
	rawgError, ok := err.(*rawg.RawgError)
	if ok {
		switch rawgError.HttpCode {
		case http.StatusNotFound:
			return "Bot not connected to servers at the moment"
		case http.StatusUnauthorized:
			return "RAWG API not availible at the moment"
		default:
			return "Unspecified HTTP error occoured"
		}
	} else {
		return "Unspecified error occoured"
	}
}
