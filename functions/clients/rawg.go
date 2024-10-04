package clients

import (
	"net/http"

	"github.com/dimuska139/rawg-sdk-go"
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
