package sms

import (
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/dysmsapi"
)

func querySendStatistics(client *dysmsapi.Client) (int64, error) {
	date := time.Now().UTC().Format("20060102")
	request := dysmsapi.CreateQuerySendStatisticsRequest()
	request.Scheme = "https"
	request.IsGlobe = requests.NewInteger(1)
	request.StartDate = date
	request.EndDate = date
	request.PageIndex = requests.NewInteger(1)
	request.PageSize = requests.NewInteger(10)
	response, err := client.QuerySendStatistics(request)
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
