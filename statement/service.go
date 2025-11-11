package statement

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/schema/services/accounting_iface/v1/accounting_ifaceconnect"
)

type statementImpl struct {
	ledger accounting_ifaceconnect.LedgerServiceClient
}

// StatementBalance implements accounting_ifaceconnect.StatementServiceHandler.
func (s *statementImpl) StatementBalance(context.Context, *connect.Request[accounting_iface.StatementBalanceRequest], *connect.ServerStream[accounting_iface.StatementBalanceResponse]) error {
	panic("unimplemented")
}

// StatementCashFlow implements accounting_ifaceconnect.StatementServiceHandler.
func (s *statementImpl) StatementCashFlow(context.Context, *connect.Request[accounting_iface.StatementCashFlowRequest]) (*connect.Response[accounting_iface.StatementCashFlowResponse], error) {
	panic("unimplemented")
}

// StatementIncome implements accounting_ifaceconnect.StatementServiceHandler.
func (s *statementImpl) StatementIncome(
	ctx context.Context,
	req *connect.Request[accounting_iface.StatementIncomeRequest],
	stream *connect.ServerStream[accounting_iface.StatementIncomeResponse]) error {

	panic("unimplemented")
}

func NewStatementService(
	ledger accounting_ifaceconnect.LedgerServiceClient,
) *statementImpl {
	return &statementImpl{
		ledger: ledger,
	}
}
