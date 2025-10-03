package accounting_core

import "github.com/dgraph-io/badger/v4"

var badgedb *badger.DB

func ConfigureAccountingCore(
	bdb *badger.DB,
) error {
	badgedb = bdb
	return nil
}
