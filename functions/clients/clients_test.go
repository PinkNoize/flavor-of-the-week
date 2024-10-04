package clients_test

import (
	"context"
	"testing"

	"github.com/PinkNoize/flavor-of-the-week/functions/clients"
)

func TestLazyLoad(t *testing.T) {
	ctx := context.Background()
	c := clients.New(ctx, "", "", "")
	_, err := c.Discord()
	if err != nil {
		t.Fatalf(`c.Discord err != nil, %v, want nil, error`, err)
	}
}
