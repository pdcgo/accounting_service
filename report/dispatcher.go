package report

import (
	"bytes"
	"context"
	"net/http"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"github.com/googleapis/gax-go/v2"
)

func NewCloudTaskReportDispatcher(
	client *cloudtasks.Client,
) ReportDispatcher {
	return func(ctx context.Context, req *cloudtaskspb.CreateTaskRequest, opts ...gax.CallOption) error {
		_, err := client.CreateTask(ctx, req)
		return err
	}
}

func NewLocalReportDispatcher() ReportDispatcher {
	return func(ctx context.Context, req *cloudtaskspb.CreateTaskRequest, opts ...gax.CallOption) error {
		httpreq := req.Task.GetHttpRequest()

		htreq, err := http.NewRequest(http.MethodPost, httpreq.GetUrl(), bytes.NewBuffer(httpreq.Body))
		if err != nil {
			return nil
		}
		_, err = http.DefaultClient.Do(htreq)

		return err
	}
}
