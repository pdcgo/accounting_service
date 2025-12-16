package setup

import (
	"fmt"

	"github.com/pdcgo/accounting_service/accounting_model"
	"github.com/pdcgo/accounting_service/adjustment"
	"github.com/pdcgo/accounting_service/ads_expense"
	"github.com/pdcgo/shared/authorization"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

type RoleItem map[string]authorization_iface.RoleAddPermissionPayload

type RoleMap map[db_models.TeamType]RoleItem

var fullAccess = []*authorization_iface.RoleAddPermissionItem{
	{
		Action: authorization_iface.Create,
		Policy: authorization_iface.Allow,
	},
	{
		Action: authorization_iface.Read,
		Policy: authorization_iface.Allow,
	},
	{
		Action: authorization_iface.Update,
		Policy: authorization_iface.Allow,
	},
	{
		Action: authorization_iface.Delete,
		Policy: authorization_iface.Allow,
	},
}

func defaultRootRolePermission() RoleMap {
	roleMap := RoleMap{
		db_models.RootTeamType: {},
		db_models.AdminTeamType: RoleItem{
			"owner": authorization_iface.RoleAddPermissionPayload{
				&accounting_model.BankTransfer{}:   fullAccess,
				&accounting_model.ExpenseEntity{}:  fullAccess,
				&accounting_model.Payment{}:        fullAccess,
				&adjustment.AdjustmentAccess{}:     fullAccess,
				&ads_expense.AdsExpense{}:          fullAccess,
				&accounting_model.BankAccountV2{}:  fullAccess,
				&db_models.OweLimitConfiguration{}: fullAccess,
			},
			"admin": authorization_iface.RoleAddPermissionPayload{
				&accounting_model.BankTransfer{}:   fullAccess,
				&accounting_model.ExpenseEntity{}:  fullAccess,
				&accounting_model.Payment{}:        fullAccess,
				&adjustment.AdjustmentAccess{}:     fullAccess,
				&ads_expense.AdsExpense{}:          fullAccess,
				&accounting_model.BankAccountV2{}:  fullAccess,
				&db_models.OweLimitConfiguration{}: fullAccess,
			},
		},
		db_models.SellingTeamType: RoleItem{
			"owner": authorization_iface.RoleAddPermissionPayload{
				&db_models.OweLimitConfiguration{}: fullAccess,
			},
			"admin": authorization_iface.RoleAddPermissionPayload{
				&db_models.OweLimitConfiguration{}: fullAccess,
			},
		},
		db_models.WarehouseTeamType: RoleItem{},
	}

	return roleMap
}

func defaultRolePermission() RoleMap {
	roleMap := RoleMap{
		db_models.RootTeamType: {},
		db_models.AdminTeamType: RoleItem{
			"owner": authorization_iface.RoleAddPermissionPayload{
				&adjustment.AdjustmentAccess{}:    fullAccess,
				&ads_expense.AdsExpense{}:         fullAccess,
				&accounting_model.Payment{}:       fullAccess,
				&accounting_model.ExpenseEntity{}: fullAccess,
				&accounting_model.BankAccountV2{}: fullAccess,
				&accounting_model.BankTransfer{}:  fullAccess,
			},
			"admin": authorization_iface.RoleAddPermissionPayload{
				&adjustment.AdjustmentAccess{}:    fullAccess,
				&ads_expense.AdsExpense{}:         fullAccess,
				&accounting_model.Payment{}:       fullAccess,
				&accounting_model.ExpenseEntity{}: fullAccess,
				&accounting_model.BankAccountV2{}: fullAccess,
				&accounting_model.BankTransfer{}:  fullAccess,
			},
		},
		db_models.SellingTeamType: RoleItem{
			"owner": authorization_iface.RoleAddPermissionPayload{
				&adjustment.AdjustmentAccess{}:    fullAccess,
				&ads_expense.AdsExpense{}:         fullAccess,
				&accounting_model.Payment{}:       fullAccess,
				&accounting_model.ExpenseEntity{}: fullAccess,
				&accounting_model.BankAccountV2{}: fullAccess,
				&accounting_model.BankTransfer{}:  fullAccess,
			},
			"admin": authorization_iface.RoleAddPermissionPayload{
				&adjustment.AdjustmentAccess{}:    fullAccess,
				&ads_expense.AdsExpense{}:         fullAccess,
				&accounting_model.Payment{}:       fullAccess,
				&accounting_model.ExpenseEntity{}: fullAccess,
				&accounting_model.BankAccountV2{}: fullAccess,
				&accounting_model.BankTransfer{}:  fullAccess,
			},
			"cs": authorization_iface.RoleAddPermissionPayload{
				&ads_expense.AdsExpense{}: fullAccess,
			},
		},
		db_models.WarehouseTeamType: RoleItem{
			"owner": authorization_iface.RoleAddPermissionPayload{
				&adjustment.AdjustmentAccess{}:    fullAccess,
				&accounting_model.Payment{}:       fullAccess,
				&accounting_model.ExpenseEntity{}: fullAccess,
			},
			"admin": authorization_iface.RoleAddPermissionPayload{
				&adjustment.AdjustmentAccess{}:    fullAccess,
				&accounting_model.ExpenseEntity{}: fullAccess,
			},
		},
	}

	return roleMap
}

func RegisterPermission(db *gorm.DB, teamID uint, teamType db_models.TeamType, streamlog func(msg string)) error {
	var err error

	roleMap := defaultRolePermission()
	rootRoleMap := defaultRootRolePermission()

	roleItem, ok := roleMap[teamType]
	if !ok {
		err = fmt.Errorf("team type %s not found in role map", teamType)
		streamlog(err.Error())
		return err
	}

	domain := authorization.NewDomainV2(db, teamID)
	for roleKey, permissions := range roleItem {
		for ent := range permissions {
			streamlog(fmt.Sprintf("check %s with entity %s", roleKey, ent.GetEntityID()))
		}

		err := domain.RoleAddPermission(roleKey, permissions)
		if err != nil {
			streamlog(err.Error())
			return err
		}
	}

	rootRoleItem := rootRoleMap[teamType]
	if !ok {
		err = fmt.Errorf("team type %s not found in root role map", teamType)
		streamlog(err.Error())
		return err
	}

	for roleKey, permissions := range rootRoleItem {
		for ent := range permissions {
			streamlog(fmt.Sprintf("check %s with entity %s", roleKey, ent.GetEntityID()))
		}

		err := domain.RoleAddPermissionWithDomain(roleKey, authorization.RootDomain, permissions)
		if err != nil {
			streamlog(err.Error())
			return err
		}
	}

	return err
}
