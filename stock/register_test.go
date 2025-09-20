package stock_test

import (
	"log/slog"
	"testing"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/stock_iface/v1"
	"github.com/pdcgo/shared/pkg/debugtool"
)

func TestConnect(t *testing.T) {
	var req connect.AnyRequest = connect.NewRequest(&stock_iface.InboundAcceptRequest{
		TeamId: 1,
	})

	debugtool.LogJson(req.Any())
	slog.Error("asdasd", "asdasd", "asdasda", slog.Any("payload", req.Any()))
}
