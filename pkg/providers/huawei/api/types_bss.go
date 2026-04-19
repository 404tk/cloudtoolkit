package api

type ShowCustomerAccountBalancesResponse struct {
	AccountBalances []AccountBalance `json:"account_balances"`
}

type AccountBalance struct {
	AccountType int32 `json:"account_type"`
	Amount      any   `json:"amount"`
}
