package report

import (
	"context"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/report_iface/v1"
	"github.com/pdcgo/schema/services/report_iface/v1/report_ifaceconnect"
	"github.com/pdcgo/shared/configs"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/protobuf/encoding/protojson"
)

// DailyUpdateBalanceAsync implements report_ifaceconnect.AccountReportServiceHandler.
func (a *accountReportImpl) DailyUpdateBalanceAsync(ctx context.Context, req *connect.Request[report_iface.DailyUpdateBalanceAsyncRequest]) (*connect.Response[report_iface.DailyUpdateBalanceAsyncResponse], error) {

	content, err := protojson.Marshal(req.Msg.Req)
	if err != nil {
		return &connect.Response[report_iface.DailyUpdateBalanceAsyncResponse]{}, err
	}

	reqheaders := make(map[string]string)
	for k, v := range req.Header() {
		if len(v) > 0 {
			reqheaders[k] = v[0]
		}
	}

	// reqheaders["Content-Type"] = "application/grpc-web"
	reqheaders["Content-Type"] = "application/json"
	reqheaders["Connect-Protocol-Version"] = "1"

	if err != nil {
		return &connect.Response[report_iface.DailyUpdateBalanceAsyncResponse]{}, err
	}

	httpreq := &cloudtaskspb.Task_HttpRequest{
		HttpRequest: &cloudtaskspb.HttpRequest{
			Url:        a.accConfig.Endpoint + report_ifaceconnect.AccountReportServiceDailyUpdateBalanceProcedure,
			HttpMethod: cloudtaskspb.HttpMethod_POST,
			Headers:    reqheaders,
			Body:       content,
		},
	}

	task := cloudtaskspb.CreateTaskRequest{
		Parent: a.cfg.GetPath(configs.SlowQueue),
		Task: &cloudtaskspb.Task{
			MessageType: httpreq,
		},
	}
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(task.Task.GetHttpRequest().Headers))

	err = a.dispather(ctx, &task)
	return &connect.Response[report_iface.DailyUpdateBalanceAsyncResponse]{}, err
}
