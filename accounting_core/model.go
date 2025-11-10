package accounting_core

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/pdcgo/schema/services/accounting_iface/v1"
	"gorm.io/gorm"
)

type CoaCode int

const (
	ASSET     CoaCode = 10
	LIABILITY CoaCode = 20
	EQUITY    CoaCode = 30
	REVENUE   CoaCode = 40
	EXPENSE   CoaCode = 50
)

type BalanceType string

func (b BalanceType) DiffBalance(debit, credit float64) float64 {
	switch b {
	case CreditBalance:
		return credit - debit
	case DebitBalance:
		return debit - credit
	default:
		return 0
	}

}

const (
	CreditBalance BalanceType = "c"
	DebitBalance  BalanceType = "d"
)

type JournalEntry struct {
	ID            uint      `json:"id" gorm:"primarykey"`
	AccountID     uint      `json:"account_id"`
	TeamID        uint      `json:"team_id"`
	TransactionID uint      `json:"transaction_id"`
	CreatedByID   uint      `json:"created_by_id"`
	EntryTime     time.Time `json:"entry_time"`
	Debit         float64   `json:"debit"`
	Credit        float64   `json:"credit"`
	Desc          string    `json:"desc"`
	Rollback      bool      `json:"rollback" gorm:"index"`

	Account     *Account     `json:"account"`
	Transaction *Transaction `json:"-"`
}

type JournalEntriesList []*JournalEntry

type ChangeBalance struct {
	Account *Account
	Debit   float64
	Credit  float64
}

func (cb *ChangeBalance) Change() float64 {
	var change float64
	switch cb.Account.BalanceType {
	case DebitBalance:
		change = cb.Debit - cb.Credit
	case CreditBalance:
		change = cb.Credit - cb.Debit
	}

	// debugtool.LogJson(cb, change)

	return change
}

func (entries JournalEntriesList) AccountBalance() (map[uint]*ChangeBalance, error) {
	changemap := map[uint]*ChangeBalance{}

	for _, en := range entries {
		if en.Account == nil {
			return changemap, errors.New("please preload account in entry")
		}
		var ok bool
		var change *ChangeBalance
		change, ok = changemap[en.AccountID]
		if !ok {
			change = &ChangeBalance{
				Debit:   0,
				Credit:  0,
				Account: en.Account,
			}
			changemap[en.AccountID] = change
		}

		// if en.Rollback {
		// 	change.Debit += en.Credit
		// 	change.Credit += en.Debit
		// } else {
		// 	change.Debit += en.Debit
		// 	change.Credit += en.Credit
		// }

		change.Debit += en.Debit
		change.Credit += en.Credit

	}
	return changemap, nil
}

func (entries JournalEntriesList) AccountBalanceKey(key AccountKey) (*ChangeBalance, error) {
	accmap, err := entries.AccountBalance()
	res := ChangeBalance{
		Account: &Account{},
	}
	if err != nil {
		return &res, err
	}

	for _, ac := range accmap {
		if ac.Account.AccountKey == key {
			return ac, nil
		}
	}

	return &res, fmt.Errorf("account not found %s", key)
}

func (entries JournalEntriesList) PrintJournalEntries(db *gorm.DB) error {
	var err error
	var debit, credit float64
	fmt.Println("=== Journal Entries ===")
	for _, e := range entries {
		debit += e.Debit
		credit += e.Credit
		e.Account = &Account{}
		err = db.Model(&Account{}).First(e.Account, e.AccountID).Error
		if err != nil {
			return err
		}
		accountName := "Unknown"
		if e.Account != nil {
			accountName = fmt.Sprintf("%s TeamID %d (%s)", e.Account.AccountKey, e.Account.TeamID, e.Account.BalanceType)
		}
		fmt.Printf(
			"[%s] %d | Txn #%d | Debit: %10.2f | Credit: %10.2f | Account: %-20s | Desc: %s\n",
			e.EntryTime.Format("2006-01-02 15:04"),
			e.TeamID,
			e.TransactionID,
			e.Debit,
			e.Credit,
			accountName,
			e.Desc,
		)
	}
	fmt.Printf("==========Debit: %10.2f, Credit: %10.2f===============\n", debit, credit)
	return nil
}

type Account struct {
	ID          uint        `json:"id" gorm:"primarykey"`
	AccountKey  AccountKey  `json:"account_key" gorm:"index:team_key,unique"`
	TeamID      uint        `json:"team_id" gorm:"index:team_key,unique"`
	Coa         CoaCode     `json:"coa"`
	BalanceType BalanceType `json:"balance_type"`
	CanAdjust   bool        `json:"can_adjust"`

	Name string `json:"name"`

	Created time.Time `json:"created"`
}

// Key implements exact_one.ExactHaveKey.
func (ac *Account) Key() string {
	return fmt.Sprintf("accounting_core/%s/%d", ac.AccountKey, ac.TeamID)
}

func (ac *Account) SetAmountEntry(amount float64, entry *JournalEntry) error {
	if amount == 0 {
		return fmt.Errorf("account %s amount entry set is zero", ac.AccountKey)
	}

	amountAbs := math.Abs(amount)

	switch ac.BalanceType {
	case CreditBalance:
		if amount > 0 {
			entry.Credit = amountAbs
		}
		if amount < 0 {
			entry.Debit = amountAbs
		}
	case DebitBalance:
		if amount > 0 {
			entry.Debit = amountAbs
		}
		if amount < 0 {
			entry.Credit = amountAbs
		}
	default:
		return fmt.Errorf("account type invalid %s", ac.BalanceType)
	}

	return nil
}

type RefType string

const (
	WithdrawalRef          RefType = "wd"
	RevenueAdjustmentRef   RefType = "revenue_adjustment"
	OrderRef               RefType = "order"
	OrderReturnRef         RefType = "order_return"
	OrderProblemRef        RefType = "order_problem"
	OrderFundRef           RefType = "order_fund"
	StockAcceptRef         RefType = "stock_accept"
	StockTransferRef       RefType = "stock_transfer"
	StockTransferAcceptRef RefType = "stock_transfer_accept"
	StockReturnRef         RefType = "stock_return"
	StockAdjustmentRef     RefType = "stock_adjustment"
	ExpenseRef             RefType = "expense"
	RestockRef             RefType = "restock"
	PaymentRef             RefType = "payment"
	AdminAdjustmentRef     RefType = "admin_adjustment"
	AdsPaymentRef          RefType = "ads_payment"
	AdjustmentRef          RefType = "common_adjustment"
)

type RefData struct {
	RefType RefType
	ID      uint
}

type RefID string

func (r RefID) Extract() (*RefData, error) {
	ss := strings.Split(string(r), "#")
	idx, err := strconv.ParseUint(ss[0], 10, 64)
	if err != nil {
		return nil, err
	}
	return &RefData{
		RefType: RefType(ss[0]),
		ID:      uint(idx),
	}, nil
}

func NewRefID(data *RefData) RefID {
	return RefID(fmt.Sprintf("%s#%d", data.RefType, data.ID))
}

type StringRefData struct {
	RefType RefType
	ID      string
}

func NewStringRefID(data *StringRefData) RefID {
	return RefID(fmt.Sprintf("%s#%s", data.RefType, data.ID))
}

type Transaction struct {
	ID          uint  `json:"id" gorm:"primarykey"`
	RefID       RefID `json:"ref_id" gorm:"index:ref_unique,unique"`
	TeamID      uint  `json:"team_id"`
	CreatedByID uint  `json:"created_by_id"`
	// Type        SourceType `json:"type" gorm:"not null"`
	Desc    string    `json:"desc"`
	Created time.Time `json:"created"`
}

// lawas

type AccountingTag struct {
	ID   uint   `json:"id" gorm:"primarykey"`
	Name string `json:"name" gorm:"index:name,unique"`
}

func SanityTag(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "_")
	return name
}

type TransactionTag struct {
	TransactionID uint `json:"transaction_id" gorm:"primaryKey"`
	TagID         uint `json:"tag_id" gorm:"primaryKey"`
}

type TransactionShop struct {
	TransactionID uint `json:"transaction_id" gorm:"primaryKey"`
	ShopID        uint `json:"shop_id" gorm:"primaryKey"`
}

type TransactionCustomerService struct {
	TransactionID     uint `json:"transaction_id" gorm:"primaryKey"`
	CustomerServiceID uint `json:"customer_service_id" gorm:"primaryKey"`
}

type TransactionSupplier struct {
	TransactionID uint `json:"transaction_id" gorm:"primaryKey"`
	SupplierID    uint `json:"supplier_id" gorm:"primaryKey"`
}

type TypeLabel struct {
	ID    uint                      `json:"id" gorm:"primarykey"`
	Key   accounting_iface.LabelKey `json:"key" gorm:"index:label_unique_key,unique"`
	Label string                    `json:"label" gorm:"index:label_unique_key,unique"`
}

type TransactionTypeLabel struct {
	TransactionID uint `json:"transaction_id" gorm:"primaryKey"`
	TypeLabelID   uint `json:"type_label_id" gorm:"primaryKey"`
}
