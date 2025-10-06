package stock

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_core"
	"github.com/pdcgo/schema/services/stock_iface/v1"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

// TransferToWarehouse implements stock_ifaceconnect.StockServiceHandler.
func (s *stockServiceImpl) TransferToWarehouse(
	ctx context.Context,
	req *connect.Request[stock_iface.TransferToWarehouseRequest],
) (*connect.Response[stock_iface.TransferToWarehouseResponse], error) {
	var err error

	pay := req.Msg
	result := &stock_iface.TransferToWarehouseResponse{}

	identity := s.auth.AuthIdentityFromHeader(req.Header())
	agent := identity.Identity()
	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			&db_models.InvTransaction{}: &authorization_iface.CheckPermission{DomainID: uint(pay.FromWarehouseId), Actions: []authorization_iface.Action{authorization_iface.Create}},
		}).
		Err()

	if err != nil {
		return connect.NewResponse(result), err
	}

	teamEntries := map[uint64]accounting_core.CreateEntry{}
	teamProduct := map[uint64]TransferItemList{}

	db := s.db.WithContext(ctx)
	err = accounting_core.OpenTransaction(db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {
		for _, d := range pay.Products {
			data := d
			if teamProduct[d.TeamId] == nil {
				teamProduct[d.TeamId] = TransferItemList{}
				teamEntries[d.TeamId] = bookmng.NewCreateEntry(uint(d.TeamId), agent.IdentityID())
			}
			teamProduct[d.TeamId] = append(teamProduct[d.TeamId], data)
		}

		ref := accounting_core.NewRefID(&accounting_core.RefData{
			RefType: accounting_core.StockTransferRef,
			ID:      uint(pay.ExtTxId),
		})

		tran := accounting_core.Transaction{
			RefID:       ref,
			TeamID:      uint(pay.FromWarehouseId),
			CreatedByID: agent.IdentityID(),
			Desc:        fmt.Sprintf("transfer stock %s", ref),
			Created:     time.Now(),
		}

		err = bookmng.
			NewTransaction().
			Create(&tran).
			Err()

		if err != nil {
			return err
		}

		// source ware
		fromware := bookmng.NewCreateEntry(uint(pay.FromWarehouseId), agent.IdentityID())
		// toware := accounting_core.NewCreateEntry(tx, uint(pay.ToWarehouseId), agent.IdentityID())

		for teamID, products := range teamProduct {
			totalAmount := products.GetTotalAmount()

			fromware.
				From(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.StockReadyAccount,
					TeamID: uint(teamID),
				}, totalAmount).
				To(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.StockTransferAccount,
					TeamID: uint(pay.ToWarehouseId),
				}, totalAmount)

			teamEntry := teamEntries[teamID]
			teamEntry.
				From(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.StockReadyAccount,
					TeamID: uint(pay.FromWarehouseId),
				}, totalAmount).
				To(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.StockTransferAccount,
					TeamID: uint(pay.ToWarehouseId),
				}, totalAmount)
		}

		err = fromware.
			Transaction(&tran).
			Commit().
			Err()

		if err != nil {
			return err
		}

		for _, entry := range teamEntries {
			err = entry.
				Transaction(&tran).
				Commit().
				Err()
			if err != nil {
				return err
			}
		}

		return nil
	})

	return connect.NewResponse(result), err
}

// TransferToWarehouseCancel implements stock_ifaceconnect.StockServiceHandler.
func (s *stockServiceImpl) TransferToWarehouseCancel(
	ctx context.Context,
	req *connect.Request[stock_iface.TransferToWarehouseCancelRequest],
) (*connect.Response[stock_iface.TransferToWarehouseCancelResponse], error) {
	var err error

	pay := req.Msg
	result := &stock_iface.TransferToWarehouseCancelResponse{}

	identity := s.auth.AuthIdentityFromHeader(req.Header())
	agent := identity.Identity()
	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			&db_models.InvTransaction{}: &authorization_iface.CheckPermission{DomainID: uint(pay.FromWarehouseId), Actions: []authorization_iface.Action{authorization_iface.Update}},
		}).
		Err()

	if err != nil {
		return connect.NewResponse(result), err
	}

	db := s.db.WithContext(ctx)
	err = db.Transaction(func(tx *gorm.DB) error {
		ref := accounting_core.NewRefID(&accounting_core.RefData{
			RefType: accounting_core.StockTransferRef,
			ID:      uint(pay.ExtTxId),
		})
		err = accounting_core.
			NewTransactionMutation(tx).
			ByRefID(ref, true).
			RollbackEntry(agent.IdentityID(), fmt.Sprintf("cancel transfer %s", ref)).
			Err()

		return err
	})

	return connect.NewResponse(result), err
}

// TransferToWarehouseAccept implements stock_ifaceconnect.StockServiceHandler.
func (s *stockServiceImpl) TransferToWarehouseAccept(
	ctx context.Context,
	req *connect.Request[stock_iface.TransferToWarehouseAcceptRequest],
) (*connect.Response[stock_iface.TransferToWarehouseAcceptResponse], error) {
	var err error

	pay := req.Msg
	result := &stock_iface.TransferToWarehouseAcceptResponse{}

	identity := s.auth.AuthIdentityFromHeader(req.Header())
	agent := identity.Identity()
	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			&db_models.InvTransaction{}: &authorization_iface.CheckPermission{DomainID: uint(pay.FromWarehouseId), Actions: []authorization_iface.Action{authorization_iface.Update}},
		}).
		Err()

	if err != nil {
		return connect.NewResponse(result), err
	}

	db := s.db.WithContext(ctx)

	teamEntries := map[uint64]accounting_core.CreateEntry{}
	teamProduct := map[uint64]TransferItemList{}
	err = accounting_core.OpenTransaction(db, func(tx *gorm.DB, bookmng accounting_core.BookManage) error {
		for _, d := range pay.Products {
			data := d
			if teamProduct[d.TeamId] == nil {
				teamProduct[d.TeamId] = TransferItemList{}
				teamEntries[d.TeamId] = bookmng.NewCreateEntry(uint(d.TeamId), agent.IdentityID())
			}
			teamProduct[d.TeamId] = append(teamProduct[d.TeamId], data)
		}

		ref := accounting_core.NewRefID(&accounting_core.RefData{
			RefType: accounting_core.StockTransferAcceptRef,
			ID:      uint(pay.ExtTxId),
		})

		tran := accounting_core.Transaction{
			RefID:       ref,
			TeamID:      uint(pay.FromWarehouseId),
			CreatedByID: agent.IdentityID(),
			Desc:        fmt.Sprintf("accept transfer %s", ref),
			Created:     time.Now(),
		}

		err = bookmng.
			NewTransaction().
			Create(&tran).
			Err()

		if err != nil {
			return err
		}

		// transfer acccept
		toware := bookmng.NewCreateEntry(uint(pay.ToWarehouseId), agent.IdentityID())

		for teamID, products := range teamProduct {
			totalAmount := products.GetTotalAmount()

			toware.
				From(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.StockTransferAccount,
					TeamID: uint(pay.FromWarehouseId),
				}, totalAmount).
				To(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.StockReadyAccount,
					TeamID: uint(teamID),
				}, totalAmount)

			teamEntry := teamEntries[teamID]
			teamEntry.
				From(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.StockTransferAccount,
					TeamID: uint(pay.ToWarehouseId),
				}, totalAmount).
				To(&accounting_core.EntryAccountPayload{
					Key:    accounting_core.StockReadyAccount,
					TeamID: uint(pay.ToWarehouseId),
				}, totalAmount)
		}

		err = toware.
			Transaction(&tran).
			Commit().
			Err()

		if err != nil {
			return err
		}

		for _, entry := range teamEntries {
			err = entry.
				Transaction(&tran).
				Desc(fmt.Sprintf("accept transfer stock %s", tran.RefID)).
				Commit().
				Err()
			if err != nil {
				return err
			}
		}

		return nil
	})

	return connect.NewResponse(result), err
}

type TransferItemList []*stock_iface.TransferItem

func (l TransferItemList) GetTotalAmount() float64 {
	var amount float64
	for _, d := range l {
		amount += d.ItemPrice * float64(d.Count)
	}

	return amount
}
