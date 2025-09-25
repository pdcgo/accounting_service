package main

import (
	"log"

	taskspb "cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/stock_iface/v1"
	"github.com/pdcgo/schema/services/stock_iface/v1/stock_ifaceconnect"
	"google.golang.org/protobuf/proto"
)

func main() {

	pay := &stock_iface.InboundCreateRequest{
		TeamId:        99,
		WarehouseId:   99,
		ExtTxId:       1,
		Source:        stock_iface.InboundSource_INBOUND_SOURCE_RESTOCK,
		PaymentMethod: stock_iface.PaymentMethod_PAYMENT_METHOD_CASH,
		ShippingFee:   1233,
		Products: []*stock_iface.VariantItem{
			{
				VariantId: 1,
				Count:     123,
				ItemPrice: 2323,
			},
		},
	}
	msg := connect.NewRequest(pay)

	rawdata, err := proto.Marshal(pay)
	if err != nil {
		panic(err)
	}

	// var client stock_ifaceconnect.StockServiceClient = &mClient{}

	// stock_iface.
	log.Println(stock_ifaceconnect.StockServiceInboundCreateProcedure)
	log.Println(msg)
	queuePath := "projects/pdcgudang/locations/asia-southeast2/queues/warehouse-event-task"

	req := taskspb.CreateTaskRequest{
		Parent: queuePath,
		Task: &taskspb.Task{
			MessageType: &taskspb.Task_HttpRequest{
				HttpRequest: &taskspb.HttpRequest{
					Url:        "" + stock_ifaceconnect.StockServiceInboundCreateProcedure,
					HttpMethod: taskspb.HttpMethod_POST,
					Headers: map[string]string{
						"Content-Type": "application/proto",
					},
					Body: rawdata,
				},
			},
		},
	}

	// client, err = cloudtasks.NewClient(ctx)
	// if err != nil {
	// 	panic(err)
	// }

	// _, err = client.CreateTask(ctx, &req)
	// if err != nil {
	// 	panic(err)
	// }

	log.Println(&req)
}
