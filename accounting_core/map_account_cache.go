package accounting_core

import (
	"errors"
)

type AccountCache interface {
	Get(accID uint) (*Account, error)
}

type BadgeAccountCache struct {
	// tx *gorm.DB
}

// Get implements AccountCache.
func (b *BadgeAccountCache) Get(accID uint) (*Account, error) {
	if badgedb == nil {
		return nil, errors.New("badgedb is not registering")
	}

	panic("unimplemented")
}
