package revenue

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/schema/services/revenue_iface/v1"
	"github.com/pdcgo/schema/services/revenue_iface/v1/revenue_ifaceconnect"
	"github.com/pdcgo/shared/configs"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/gorm"
)

// OrderReturnAsync implements revenue_ifaceconnect.RevenueServiceHandler.
func (r *revenueServiceImpl) OrderReturnAsync(
	ctx context.Context,
	req *connect.Request[revenue_iface.OrderReturnAsyncRequest],
) (*connect.Response[revenue_iface.OrderReturnAsyncResponse], error) {
	content, err := protojson.Marshal(req.Msg.Data)
	if err != nil {
		return &connect.Response[revenue_iface.OrderReturnAsyncResponse]{}, err
	}

	headers := req.Header()
	hh := propagation.HeaderCarrier(headers)
	otel.GetTextMapPropagator().Inject(ctx, hh)

	reqheaders := make(map[string]string)
	for k, v := range headers {
		if len(v) > 0 {
			reqheaders[k] = v[0]
		}
	}

	// reqheaders["Content-Type"] = "application/grpc-web"
	reqheaders["Content-Type"] = "application/json"
	reqheaders["Connect-Protocol-Version"] = "1"

	httpreq := &cloudtaskspb.Task_HttpRequest{
		HttpRequest: &cloudtaskspb.HttpRequest{
			Url:        r.accountingServiceConfig.Endpoint + revenue_ifaceconnect.RevenueServiceOrderReturnProcedure,
			HttpMethod: cloudtaskspb.HttpMethod_POST,
			Headers:    reqheaders,
			Body:       content,
		},
	}

	task := cloudtaskspb.CreateTaskRequest{
		Parent: r.cfg.GetPath(configs.SlowQueue),
		Task: &cloudtaskspb.Task{
			MessageType: httpreq,
		},
	}

	err = r.dispatcher(ctx, &task)
	return &connect.Response[revenue_iface.OrderReturnAsyncResponse]{}, err
}

// OrderReturn implements revenue_ifaceconnect.RevenueServiceHandler.
func (r *revenueServiceImpl) OrderReturn(
	ctx context.Context,
	req *connect.Request[revenue_iface.OrderReturnRequest],
) (*connect.Response[revenue_iface.OrderReturnResponse], error) {
	var err error

	result := revenue_iface.OrderReturnResponse{}
	pay := req.Msg

	// var domainCheck uint
	// switch pay.RequestFrom {
	// case common.RequestFrom_REQUEST_FROM_ADMIN:
	// 	domainCheck = authorization.RootDomain
	// case common.RequestFrom_REQUEST_FROM_SELLING:
	// 	domainCheck = uint(pay.TeamId)
	// case common.RequestFrom_REQUEST_FROM_WAREHOUSE:
	// 	domainCheck = uint(pay.WarehouseId)
	// default:
	// 	domainCheck = uint(pay.TeamId)
	// }

	identity := r.auth.AuthIdentityFromHeader(req.Header())
	agent := identity.Identity()
	// identity.
	// 	HasPermission(authorization_iface.CheckPermissionGroup{
	// 		&db_models.Order{}: &authorization_iface.CheckPermission{
	// 			DomainID: domainCheck,
	// 			Actions:  []authorization_iface.Action{authorization_iface.Update},
	// 		},
	// 	})

	err = identity.Err()
	if err != nil {
		return connect.NewResponse(&result), err
	}

	// order label
	orderInfo := pay.OrderInfo
	descLabel := fmt.Sprintf("resi: %s orderid: %s", orderInfo.Receipt, orderInfo.ExternalOrderId)

	db := r.db.WithContext(ctx)
	err = accounting_core.OpenTransaction(ctx, db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {
		ref := accounting_core.NewRefID(&accounting_core.RefData{
			RefType: accounting_core.OrderReturnRef,
			ID:      uint(pay.OrderId),
		})
		tran := accounting_core.Transaction{
			RefID:       ref,
			TeamID:      uint(pay.TeamId),
			CreatedByID: agent.IdentityID(),
			Desc:        fmt.Sprintf("returning order %s %s", ref, descLabel),
			Created:     time.Now(),
		}

		err = bookmng.
			NewTransaction().
			Create(&tran).
			AddCustomerServiceID(uint(pay.LabelInfo.CsId)).
			AddShopID(uint(pay.LabelInfo.ShopId)).
			AddTypeLabel(pay.LabelInfo.TypeLabels).
			Err()

		if err != nil {
			return err
		}

		// entry selling
		entry := bookmng.
			NewCreateEntry(uint(pay.TeamId), agent.IdentityID()).
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.SellingReceivableAccount,
				TeamID: uint(pay.TeamId),
			}, pay.OrderAmount).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.SellingReturnExpenseAccount,
				TeamID: uint(pay.TeamId),
			}, pay.OrderAmount).
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.StockCostAccount,
				TeamID: uint(pay.WarehouseId),
			}, pay.StockAmount).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.StockPendingAccount,
				TeamID: uint(pay.WarehouseId),
			}, pay.StockAmount)

		err = entry.
			Transaction(&tran).
			Commit().
			Err()

		if err != nil {
			return err
		}

		// entry gudang
		entry = bookmng.
			NewCreateEntry(uint(pay.WarehouseId), agent.IdentityID()).
			From(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.StockCostAccount,
				TeamID: uint(pay.TeamId),
			}, pay.StockAmount).
			To(&accounting_core.EntryAccountPayload{
				Key:    accounting_core.StockPendingAccount,
				TeamID: uint(pay.TeamId),
			}, pay.StockAmount)

		err = entry.
			Transaction(&tran).
			Commit().
			Err()

		if err != nil {
			return err
		}
		return nil
	})

	return connect.NewResponse(&result), err
}
