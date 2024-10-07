package clients

import (
	"context"
	"net/http"

	"github.com/dimuska139/rawg-sdk-go/v3"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

const STEAM_STORE int = 1
const GOG_STORE int = 5
const EPIC_GAMES int = 11

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
	ctxzap.Info(ctx, "RAWG: Getting game", zap.String("name", game), zap.String("type", "GetGame"))
	return r.client.GetGame(ctx, game)
}

func (r *Rawg) SearchGame(ctx context.Context, name string, page, pageSize int) ([]*rawg.Game, int, error) {
	ctxzap.Info(ctx, "RAWG: Searching games", zap.String("name", name), zap.String("type", "GetGames"))
	return r.client.GetGames(ctx, rawg.NewGamesFilter().SetPageSize(pageSize).SetPage(page+1).SetStores(STEAM_STORE, GOG_STORE, EPIC_GAMES).SetSearch(name))
}
