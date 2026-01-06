package sas

import (
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/sas"
)

var eventStatus = map[int]string{
	1:  "待处理",
	2:  "已忽略",
	4:  "已确认",
	8:  "已标记误报",
	16: "处理中",
	32: "处理完毕",
	64: "已经过期",
}

type Driver struct {
	Cred *credentials.StsTokenCredential
}

func (d *Driver) NewClient() (*sas.Client, error) {
	return sas.NewClientWithOptions("cn-hangzhou", sdk.NewConfig(), d.Cred)
}

func (d *Driver) DumpEvents() ([]schema.Event, error) {
	var events []schema.Event
	client, err := d.NewClient()
	if err != nil {
		return events, err
	}
	request := sas.CreateDescribeSuspEventsRequest()
	request.Scheme = "https"
	/*
		// The filtering result of the specified source IP address is invalid
		switch sourceIp {
		case "all":
		case "self":
			ip, err := utils.HttpGet(utils.IpInfo)
			if err != nil {
				logger.Error(err)
				return
			}
			logger.Info("Current export IP:", string(ip))
			request.SourceIp = string(ip)
		default:
			request.SourceIp = sourceIp
		}
	*/

	response, err := client.DescribeSuspEvents(request)
	if err != nil {
		return events, err
	}
	for _, event := range response.SuspEvents {
		_event := schema.Event{
			Id:       event.SecurityEventIds,
			Name:     event.AlarmEventNameDisplay,
			Affected: event.InstanceName,
			Status:   eventStatus[event.EventStatus],
			Time:     event.LastTime,
		}
		for _, detail := range event.Details {
			switch detail.NameDisplay {
			case "调用的API":
				_event.API = detail.ValueDisplay
			case "调用IP", "登录源IP":
				_event.SourceIp = detail.ValueDisplay
			case "AK":
				_event.AccessKey = detail.ValueDisplay
			default:
			}
		}
		events = append(events, _event)
	}
	return events, nil

}

func (d *Driver) HandleEvents(eid string) {
	client, err := d.NewClient()
	if err != nil {
		logger.Error(err)
		return
	}
	request := requests.NewCommonRequest()
	request.Method = "POST"
	request.Scheme = "https"
	request.Domain = "tds.aliyuncs.com"
	request.Version = "2018-12-03"
	request.ApiName = "HandleSecurityEvents"
	request.QueryParams["OperationCode"] = "advance_mark_mis_info"
	ids := strings.Split(eid, ",")
	for i, id := range ids {
		k := fmt.Sprintf("SecurityEventIds.%d", i+1)
		v := strings.TrimSpace(id)
		request.QueryParams[k] = v
	}
	// MarkMissParam not work, looks like an api bug
	// request.QueryParams["MarkMissParam"] = "[{\"uuid\":\"ALL\",\"field\":\"tool_name\",\"operate\":\"strEqual\",\"fieldValue\":\"cloudtoolkit\"}]"
	request.QueryParams["MarkBatch"] = "true"
	response, err := client.ProcessCommonRequest(request)
	if err != nil {
		logger.Error(err)
		return
	}
	fmt.Println(response.GetHttpContentString())
}
