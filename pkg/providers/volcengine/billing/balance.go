package billing

import (
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/volcengine/volcengine-go-sdk/service/billing"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"github.com/volcengine/volcengine-go-sdk/volcengine/session"
)

func QueryAccountBalance(conf *volcengine.Config) {
	sess, _ := session.NewSession(conf.WithRegion("cn-beijing"))
	svc := billing.New(sess)
	queryBalanceAcctInput := &billing.QueryBalanceAcctInput{}
	resp, err := svc.QueryBalanceAcct(queryBalanceAcctInput)
	if err == nil {
		logger.Warning("Available cash amount:", *resp.AvailableBalance)
	}
}
