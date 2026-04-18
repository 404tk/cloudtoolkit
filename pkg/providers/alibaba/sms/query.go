package sms

import (
	"context"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
)

func (d *Driver) querySendStatistics(ctx context.Context, client *api.Client, region string) (int64, error) {
	now := time.Now
	if d.now != nil {
		now = d.now
	}
	date := now().UTC().Format("20060102")
	response, err := client.QuerySMSSendStatistics(ctx, region, date)
	if err != nil {
		return 0, err
	}
	return response.Data.TotalSize, nil
}

/*
func querySendDetails(client *dysmsapi.Client, phone string) {
	request := dysmsapi.CreateQuerySendDetailsRequest()
	request.Scheme = "https"
	request.SendDate = time.Now().UTC().Format("20060102")
	request.PageSize = requests.NewInteger(10)
	request.CurrentPage = requests.NewInteger(1)
	request.PhoneNumber = phone
	response, err := client.QuerySendDetails(request)
	if err != nil {
		logger.Error(err)
		return
	}
	fmt.Printf("\n%-10s\t%-90s\n", "SendDate", "Content")
	fmt.Printf("%-10s\t%-90s\n", "--------", "-------")
	for _, detail := range response.SmsSendDetailDTOs.SmsSendDetailDTO {
		fmt.Printf("%-10s\t%-90s\n\n", detail.SendDate, detail.Content)
	}
}
*/
