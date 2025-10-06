package accounting_core

import (
	"context"
	"encoding/json"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/report_iface/v1"
	"github.com/pdcgo/schema/services/report_iface/v1/report_ifaceconnect"
	"github.com/pdcgo/shared/configs"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type CustomHandler func(bookmng BookManage) error

var customHandler = map[string]CustomHandler{}

func RegisterCustomHandler(name string, handler CustomHandler) func() {
	customHandler[name] = handler
	return func() {
		delete(customHandler, name)
	}
}

func NewDailyBalanceHandler(dispatcher AccountReportServiceClientDispatcher) CustomHandler {

	return func(bookmng BookManage) error {
		label := &report_iface.TxLabelExtra{}

		blabel := bookmng.LabelExtra()
		if blabel != nil {
			tagIDs := []uint64{}

			for _, tagID := range blabel.TagIDs {
				tagIDs = append(tagIDs, uint64(tagID))
			}

			label = &report_iface.TxLabelExtra{
				ShopId:     uint64(blabel.ShopID),
				CsId:       uint64(blabel.CsID),
				TagIds:     tagIDs,
				SupplierId: uint64(blabel.SupplierID),
			}
		}

		entries := []*report_iface.EntryPayload{}
		for _, entry := range bookmng.Entries() {
			entries = append(entries, &report_iface.EntryPayload{
				Id:            uint64(entry.ID),
				TransactionId: uint64(entry.TransactionID),
				Desc:          entry.Desc,
				AccountId:     uint64(entry.AccountID),
				TeamId:        uint64(entry.TeamID),
				Debit:         entry.Debit,
				Credit:        entry.Credit,
				EntryTime:     timestamppb.New(entry.EntryTime),
			})
		}

		msg := report_iface.DailyUpdateBalanceRequest{
			Entries:    entries,
			LabelExtra: label,
		}
		req := connect.NewRequest(&msg)
		_, err := dispatcher.DailyUpdateBalance(context.Background(), req)
		return err
	}
}

type AccountReportServiceClientDispatcher report_ifaceconnect.AccountReportServiceClient

type accReportDispatcher struct {
	client *cloudtasks.Client
	cfg    *configs.DispatcherConfig
	host   string
}

// Balance implements AccountReportServiceClientDispatcher.
func (a *accReportDispatcher) Balance(context.Context, *connect.Request[report_iface.BalanceRequest]) (*connect.Response[report_iface.BalanceResponse], error) {
	panic("unimplemented")
}

// BalanceDetail implements AccountReportServiceClientDispatcher.
func (a *accReportDispatcher) BalanceDetail(context.Context, *connect.Request[report_iface.BalanceDetailRequest]) (*connect.Response[report_iface.BalanceDetailResponse], error) {
	panic("unimplemented")
}

// DailyBalance implements AccountReportServiceClientDispatcher.
func (a *accReportDispatcher) DailyBalance(context.Context, *connect.Request[report_iface.DailyBalanceRequest]) (*connect.Response[report_iface.DailyBalanceResponse], error) {
	panic("unimplemented")
}

// DailyBalanceDetail implements AccountReportServiceClientDispatcher.
func (a *accReportDispatcher) DailyBalanceDetail(context.Context, *connect.Request[report_iface.DailyBalanceDetailRequest]) (*connect.Response[report_iface.DailyBalanceDetailResponse], error) {
	panic("unimplemented")
}

// DailyUpdateBalance implements AccountReportServiceClientDispatcher.
func (a *accReportDispatcher) DailyUpdateBalance(
	ctx context.Context,
	req *connect.Request[report_iface.DailyUpdateBalanceRequest],
) (*connect.Response[report_iface.DailyUpdateBalanceResponse], error) {
	content, err := json.Marshal(req.Msg)
	if err != nil {
		return &connect.Response[report_iface.DailyUpdateBalanceResponse]{}, err
	}

	reqheaders := make(map[string]string)
	for k, v := range req.Header() {
		if len(v) > 0 {
			reqheaders[k] = v[0]
		}
	}

	reqheaders["Content-Type"] = "application/json"

	task := cloudtaskspb.CreateTaskRequest{
		Parent: a.cfg.GetPath(configs.SlowQueue),
		Task: &cloudtaskspb.Task{
			MessageType: &cloudtaskspb.Task_HttpRequest{
				HttpRequest: &cloudtaskspb.HttpRequest{
					Url:        a.host + report_ifaceconnect.AccountReportServiceDailyUpdateBalanceProcedure,
					HttpMethod: cloudtaskspb.HttpMethod_POST,
					Headers:    reqheaders,
					Body:       content,
				},
			},
		},
	}

	_, err = a.client.CreateTask(ctx, &task)
	return &connect.Response[report_iface.DailyUpdateBalanceResponse]{}, err
}

func NewAccountReportServiceClientDispatcher(
	client *cloudtasks.Client,
	cfg *configs.AppConfig,
) AccountReportServiceClientDispatcher {
	return &accReportDispatcher{
		client: client,
		host:   cfg.AccountingService.Endpoint,
		cfg:    &cfg.DispatcherConfig,
	}
}
