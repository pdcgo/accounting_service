package accounting_core

type AccountKey string

const (
	SuplierCashAccount       AccountKey = "supplier_cash"
	SuplierReceivableAccount AccountKey = "supplier_receivable"

	StockReadyAccount AccountKey = "stock_ready"

	StockCrossAccount      AccountKey = "stock_cross"
	SupplierPayableAccount AccountKey = "supplier_payable"
	ShippingPayableAccount AccountKey = "shipping_payable"

	StockCrossReceivableAccount AccountKey = "stock_cross_receivable"
	StockCrossPayableAccount    AccountKey = "stock_cross_payable"

	DebtAccount         AccountKey = "debt"
	OtherExpenseAccount AccountKey = "other_expense"
	CapitalStartAccount AccountKey = "capital_start"
	BankFeeAccount      AccountKey = "bank_fee"
)

// asset
const (
	CashAccount                  AccountKey = "cash"
	ShopeepayAccount             AccountKey = "shopeepay"
	SellingEstReceivableAccount  AccountKey = "selling_est_receivable"
	SellingAdjReceivableAccount  AccountKey = "selling_adj_receivable"
	SellingReceivableAccount     AccountKey = "selling_receivable"
	StockPendingAccount          AccountKey = "stock_pending"
	StockTransferAccount         AccountKey = "stock_transfer"
	StockLostAccount             AccountKey = "stock_lost"
	StockBrokenAccount           AccountKey = "stock_broken"
	StockPendingFeeAccount       AccountKey = "stock_pending_fee"
	StockCodFeeAccount           AccountKey = "stock_cod_fee"
	ReceivableAccount            AccountKey = "receivable"
	PendingPaymentReceiveAccount AccountKey = "pending_payment_receive"
	PendingPaymentPayAccount     AccountKey = "pending_payment_pay"
	// PaymentInTransitAccount      AccountKey = "payment_in_transit"
	AdjAssetAccount AccountKey = "adj_asset"
)

// Equity
const (
	AdjEquityAccount AccountKey = "adj_equity"
)

// liability
const (
	PayableAccount      AccountKey = "payable"
	AdjLiabilityAccount AccountKey = "adj_liability"
)

// expense
const (
	CodCostAccount            AccountKey = "cod_cost_account"
	StockCostAccount          AccountKey = "stock_cost_account"
	StockBorrowCostAmount     AccountKey = "stock_borrow_cost"
	StockBrokenCostAccount    AccountKey = "stock_broken_expense"
	StockLostCostAccount      AccountKey = "stock_lost_expense"
	WarehouseCostAccount      AccountKey = "warehouse_cost"
	SalaryAccount             AccountKey = "salary"
	InternetConnectionAccount AccountKey = "internet_connection"
	PackingExpenseAccount     AccountKey = "packing_cost"
	ElectricityExpenseAccount AccountKey = "electricity"
	KitchenExpenseAccount     AccountKey = "kitchen_expense"
	EquipmentExpenseAccount   AccountKey = "equipment_expense"
	ToolExpenseAccount        AccountKey = "tool_expense"
	TransportExpenseAccount   AccountKey = "transport"
	OwnerAccommodationAccount AccountKey = "owner_accomodation"
	ServerExpenseAccount      AccountKey = "server"
	ShippingExpenseAccount    AccountKey = "shipping_expense"
	AdsExpenseAccount         AccountKey = "ads_expense"
	AdjExpenseAccount         AccountKey = "adj_expense"
	// ongkir ?
	// om + harum ?

)

// revenue
const (
	ServiceRevenueAccount         AccountKey = "service_revenue"
	BorrowStockRevenueAccount     AccountKey = "borrow_stock_revenue"
	SalesRevenueAdjustmentAccount AccountKey = "sales_revenue_adjustment"
	SalesRevenueAccount           AccountKey = "sales_revenue"
	SalesReturnRevenueAccount     AccountKey = "sales_return_revenue"
	AdjRevenueAccount             AccountKey = "adj_revenue"
)

type AccountKeyInfo struct {
	AccountKey  AccountKey  `json:"key" gorm:"primaryKey"`
	Coa         CoaCode     `json:"coa"`
	BalanceType BalanceType `json:"account_type"`
}

func DefaultSeedAccount() []*Account {
	return []*Account{
		// revenue
		{
			AccountKey:  ServiceRevenueAccount,
			Coa:         REVENUE,
			BalanceType: CreditBalance,
		},
		{
			AccountKey:  SalesRevenueAdjustmentAccount,
			Coa:         REVENUE,
			BalanceType: CreditBalance,
		},
		{
			AccountKey:  SalesRevenueAccount,
			Coa:         REVENUE,
			BalanceType: CreditBalance,
		},
		{
			AccountKey:  SalesReturnRevenueAccount,
			Coa:         REVENUE,
			BalanceType: CreditBalance,
		},
		{
			AccountKey:  BorrowStockRevenueAccount,
			Coa:         REVENUE,
			BalanceType: CreditBalance,
		},
		// asset
		{
			AccountKey:  PendingPaymentPayAccount,
			Coa:         ASSET,
			BalanceType: DebitBalance,
		},

		{
			AccountKey:  PendingPaymentReceiveAccount,
			Coa:         ASSET,
			BalanceType: DebitBalance,
		},
		{
			AccountKey:  ShopeepayAccount,
			Coa:         ASSET,
			BalanceType: DebitBalance,
		},
		{
			AccountKey:  ReceivableAccount,
			Coa:         ASSET,
			BalanceType: DebitBalance,
		},
		{
			AccountKey:  SellingEstReceivableAccount,
			Coa:         ASSET,
			BalanceType: DebitBalance,
		},
		{
			AccountKey:  SellingAdjReceivableAccount,
			Coa:         ASSET,
			BalanceType: DebitBalance,
		},

		{
			AccountKey:  SellingReceivableAccount,
			Coa:         ASSET,
			BalanceType: DebitBalance,
		},
		{
			AccountKey:  StockLostAccount,
			Coa:         ASSET,
			BalanceType: DebitBalance,
		},
		{
			AccountKey:  StockTransferAccount,
			Coa:         ASSET,
			BalanceType: DebitBalance,
		},
		{
			AccountKey:  StockPendingAccount,
			Coa:         ASSET,
			BalanceType: DebitBalance,
		},
		{
			AccountKey:  StockBrokenAccount,
			Coa:         ASSET,
			BalanceType: DebitBalance,
		},
		{
			AccountKey:  StockPendingFeeAccount,
			Coa:         ASSET,
			BalanceType: DebitBalance,
		},
		{
			AccountKey:  StockCodFeeAccount,
			Coa:         ASSET,
			BalanceType: DebitBalance,
		},
		{
			AccountKey:  StockReadyAccount,
			Coa:         ASSET,
			BalanceType: DebitBalance,
		},
		// expense
		{
			AccountKey:  AdsExpenseAccount,
			Coa:         EXPENSE,
			BalanceType: DebitBalance,
		},
		{
			AccountKey:  CodCostAccount,
			Coa:         EXPENSE,
			BalanceType: DebitBalance,
		},
		{
			AccountKey:  WarehouseCostAccount,
			Coa:         EXPENSE,
			BalanceType: DebitBalance,
		},
		{
			AccountKey:  ShippingExpenseAccount,
			Coa:         EXPENSE,
			BalanceType: DebitBalance,
		},
		{
			AccountKey:  StockCostAccount,
			Coa:         EXPENSE,
			BalanceType: DebitBalance,
		},
		{
			AccountKey:  StockLostCostAccount,
			Coa:         EXPENSE,
			BalanceType: DebitBalance,
		},
		{
			AccountKey:  StockBrokenCostAccount,
			Coa:         EXPENSE,
			BalanceType: DebitBalance,
		},
		{
			AccountKey:  StockBorrowCostAmount,
			Coa:         EXPENSE,
			BalanceType: DebitBalance,
		},

		// liability
		{
			AccountKey:  PayableAccount,
			Coa:         LIABILITY,
			BalanceType: CreditBalance,
		},

		// general
		{
			AccountKey:  CashAccount,
			Coa:         ASSET,
			BalanceType: DebitBalance,
		},
		{
			AccountKey:  OwnerAccommodationAccount,
			Coa:         EXPENSE,
			BalanceType: DebitBalance,
		},
		{
			AccountKey:  SalaryAccount,
			Coa:         EXPENSE,
			BalanceType: DebitBalance,
		},
		{
			AccountKey:  InternetConnectionAccount,
			Coa:         EXPENSE,
			BalanceType: DebitBalance,
		},
		{
			AccountKey:  PackingExpenseAccount,
			Coa:         EXPENSE,
			BalanceType: DebitBalance,
		},
		{
			AccountKey:  ElectricityExpenseAccount,
			Coa:         EXPENSE,
			BalanceType: DebitBalance,
		},
		{
			AccountKey:  KitchenExpenseAccount,
			Coa:         EXPENSE,
			BalanceType: DebitBalance,
		},
		{
			AccountKey:  EquipmentExpenseAccount,
			Coa:         EXPENSE,
			BalanceType: DebitBalance,
		},
		{
			AccountKey:  ToolExpenseAccount,
			Coa:         EXPENSE,
			BalanceType: DebitBalance,
		},
		{
			AccountKey:  TransportExpenseAccount,
			Coa:         EXPENSE,
			BalanceType: DebitBalance,
		},
		{
			AccountKey:  ServerExpenseAccount,
			Coa:         EXPENSE,
			BalanceType: DebitBalance,
		},
		{
			AccountKey:  KitchenExpenseAccount,
			Coa:         EXPENSE,
			BalanceType: DebitBalance,
		},
		{
			AccountKey:  OtherExpenseAccount,
			Coa:         EXPENSE,
			BalanceType: DebitBalance,
		},
		{
			AccountKey:  BankFeeAccount,
			Coa:         EXPENSE,
			BalanceType: DebitBalance,
		},
	}
}

// var ChartOfAccounts = []Account{
// 	// Assets
// 	{Code: "1000", Name: "Assets", Type: Asset},
// 	{Code: "1100", Name: "Current Assets", Type: Asset, Parent: "1000"},
// 	{Code: "1110", Name: "Cash", Type: Asset, Parent: "1100"},
// 	{Code: "1120", Name: "Bank", Type: Asset, Parent: "1100"},
// 	{Code: "1130", Name: "Accounts Receivable", Type: Asset, Parent: "1100"},
// 	{Code: "1140", Name: "Inventory", Type: Asset, Parent: "1100"},
// 	{Code: "1150", Name: "Prepaid Expenses", Type: Asset, Parent: "1100"},
// 	{Code: "1200", Name: "Fixed Assets", Type: Asset, Parent: "1000"},
// 	{Code: "1210", Name: "Equipment", Type: Asset, Parent: "1200"},
// 	{Code: "1220", Name: "Buildings", Type: Asset, Parent: "1200"},
// 	{Code: "1230", Name: "Vehicles", Type: Asset, Parent: "1200"},
// 	{Code: "1240", Name: "Accumulated Depreciation", Type: Asset, Parent: "1200"},

// 	// Liabilities
// 	{Code: "2000", Name: "Liabilities", Type: Liability},
// 	{Code: "2100", Name: "Current Liabilities", Type: Liability, Parent: "2000"},
// 	{Code: "2110", Name: "Accounts Payable", Type: Liability, Parent: "2100"},
// 	{Code: "2120", Name: "Salaries Payable", Type: Liability, Parent: "2100"},
// 	{Code: "2130", Name: "Taxes Payable", Type: Liability, Parent: "2100"},
// 	{Code: "2200", Name: "Long-Term Liabilities", Type: Liability, Parent: "2000"},
// 	{Code: "2210", Name: "Bank Loan", Type: Liability, Parent: "2200"},
// 	{Code: "2220", Name: "Bonds Payable", Type: Liability, Parent: "2200"},

// 	// Equity
// 	{Code: "3000", Name: "Equity", Type: Equity},
// 	{Code: "3110", Name: "Owner’s Capital", Type: Equity, Parent: "3000"},
// 	{Code: "3120", Name: "Retained Earnings", Type: Equity, Parent: "3000"},
// 	{Code: "3130", Name: "Dividends / Drawings", Type: Equity, Parent: "3000"},

// 	// Revenue
// 	{Code: "4000", Name: "Revenue", Type: Revenue},
// 	{Code: "4110", Name: "Sales Revenue", Type: Revenue, Parent: "4000"},
// 	{Code: "4120", Name: "Service Revenue", Type: Revenue, Parent: "4000"},
// 	{Code: "4130", Name: "Interest Income", Type: Revenue, Parent: "4000"},

// 	// Expenses
// 	{Code: "5000", Name: "Expenses", Type: Expense},
// 	{Code: "5100", Name: "Operating Expenses", Type: Expense, Parent: "5000"},
// 	{Code: "5110", Name: "Cost of Goods Sold", Type: Expense, Parent: "5100"},
// 	{Code: "5120", Name: "Rent Expense", Type: Expense, Parent: "5100"},
// 	{Code: "5130", Name: "Salaries Expense", Type: Expense, Parent: "5100"},
// 	{Code: "5140", Name: "Utilities Expense", Type: Expense, Parent: "5100"},
// 	{Code: "5150", Name: "Advertising Expense", Type: Expense, Parent: "5100"},
// 	{Code: "5200", Name: "Administrative Expenses", Type: Expense, Parent: "5000"},
// 	{Code: "5210", Name: "Office Supplies", Type: Expense, Parent: "5200"},
// 	{Code: "5220", Name: "Depreciation Expense", Type: Expense, Parent: "5200"},
// 	{Code: "5230", Name: "Insurance Expense", Type: Expense, Parent: "5200"},
// 	{Code: "5300", Name: "Financial Expenses", Type: Expense, Parent: "5000"},
// 	{Code: "5310", Name: "Bank Charges", Type: Expense, Parent: "5300"},
// 	{Code: "5320", Name: "Interest Expense", Type: Expense, Parent: "5300"},
// }
