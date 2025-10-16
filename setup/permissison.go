package setup

import (
	"fmt"

	"github.com/pdcgo/accounting_service/accounting_model"
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

var accountingPermission authorization_iface.RoleAddPermissionPayload = authorization_iface.RoleAddPermissionPayload{
	&accounting_model.BankTransfer{}:  fullAccess,
	&accounting_model.ExpenseEntity{}: fullAccess,
	&accounting_model.Payment{}:       fullAccess,
}

func defaultRootRolePermission() RoleMap {
	roleMap := RoleMap{
		db_models.RootTeamType: {},
		db_models.AdminTeamType: RoleItem{
			"owner": accountingPermission,
			"admin": accountingPermission,
		},
		db_models.SellingTeamType:   RoleItem{},
		db_models.WarehouseTeamType: RoleItem{},
	}

	return roleMap
}

func defaultRolePermission() RoleMap {
	roleMap := RoleMap{
		db_models.RootTeamType: {},
		db_models.AdminTeamType: RoleItem{
			"owner": accountingPermission,
			"admin": accountingPermission,
		},
		db_models.SellingTeamType: RoleItem{
			"owner": accountingPermission,
			"admin": accountingPermission,
		},
		db_models.WarehouseTeamType: RoleItem{
			"owner": accountingPermission,
			"admin": accountingPermission,
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
