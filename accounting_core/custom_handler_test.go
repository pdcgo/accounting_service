package accounting_core_test

import (
	"testing"
	"time"

	"github.com/pdcgo/schema/services/report_iface/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestModelProtoJson(t *testing.T) {
	entry := report_iface.EntryPayload{
		EntryTime: timestamppb.New(time.Unix(1, 0)),
	}

	content, err := protojson.Marshal(&entry)
	assert.Nil(t, err)

	assert.Equal(t, `{"entryTime":"1970-01-01T00:00:01Z"}`, string(content))
}
