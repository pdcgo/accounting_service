package accounting_core

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/report_iface/v1"
	"github.com/pdcgo/schema/services/report_iface/v1/report_ifaceconnect"
)

type CustomHandler func(ctx context.Context, msg *report_iface.DailyUpdateBalanceRequest) error

var customHandler = map[string]CustomHandler{}

func RegisterCustomHandler(name string, handler CustomHandler) func() {
	customHandler[name] = handler
	return func() {
		delete(customHandler, name)
	}
}

func NewDailyBalanceHandler(dispatcher report_ifaceconnect.AccountReportServiceClient) CustomHandler {

	return func(ctx context.Context, msg *report_iface.DailyUpdateBalanceRequest) error {
		_, err := dispatcher.DailyUpdateBalanceAsync(ctx, &connect.Request[report_iface.DailyUpdateBalanceAsyncRequest]{
			Msg: &report_iface.DailyUpdateBalanceAsyncRequest{
				Req: msg,
			},
		})
		return err
	}
}
