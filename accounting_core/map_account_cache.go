package accounting_core

type AccountCache interface {
	Get(accID uint) (*Account, error)
}
