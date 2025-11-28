package account

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/accounting_service/accounting_model"
	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

// AccountCreate implements accounting_ifaceconnect.AccountServiceHandler.
func (a *accountServiceImpl) AccountCreate(
	ctx context.Context,
	req *connect.Request[accounting_iface.AccountCreateRequest],
) (*connect.Response[accounting_iface.AccountCreateResponse], error) {
	var err error
	res := connect.NewResponse(&accounting_iface.AccountCreateResponse{})

	pay := req.Msg
	identity := a.
		auth.
		AuthIdentityFromHeader(req.Header())
	err = identity.
		HasPermission(authorization_iface.CheckPermissionGroup{
			&accounting_model.BankAccountV2{}: &authorization_iface.CheckPermission{
				DomainID: uint(pay.TeamId),
				Actions:  []authorization_iface.Action{authorization_iface.Create},
			},
		}).
		Err()

	if err != nil {
		return res, err
	}

	err = a.
		db.
		WithContext(ctx).
		Transaction(func(tx *gorm.DB) error {
			acc := accounting_model.BankAccountV2{
				TeamID:        uint(pay.TeamId),
				Name:          pay.Name,
				NumberID:      pay.NumberId,
				AccountTypeID: uint(pay.AccountTypeId),
				CreatedAt:     time.Now(),
			}
			err = tx.Save(&acc).Error
			res.Msg.Id = uint64(acc.ID)
			if err != nil {
				if errors.Is(err, gorm.ErrDuplicatedKey) {
					return errors.New("account duplicated")
				}
				return err
			}

			if len(pay.Labels) != 0 {
				for _, label := range pay.Labels {
					lab := accounting_model.BankAccountLabel{
						Key:   label.Key,
						Value: label.Name,
					}

					err = tx.Where("key = ?", lab.Key).Find(&lab).Error
					if err != nil {
						return err
					}
					if lab.ID == 0 {
						err = tx.Save(&lab).Error
						if err != nil {
							return err
						}
					}

					rel := accounting_model.BankAccountLabelRelation{
						AccountID: acc.ID,
						LabelID:   lab.ID,
					}

					err = tx.Save(&rel).Error
					if err != nil {
						return err
					}
				}

			}

			return err
		})

	if err != nil {
		return res, err
	}

	return res, nil

}
