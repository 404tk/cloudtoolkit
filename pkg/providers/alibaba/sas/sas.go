package sas

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	aliauth "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
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
	Cred          aliauth.Credential
	clientOptions []api.Option
}

func (d *Driver) newClient() *api.Client {
	return api.NewClient(d.Cred, d.clientOptions...)
}

func (d *Driver) SetClientOptions(opts ...api.Option) {
	d.clientOptions = append([]api.Option(nil), opts...)
}

func (d *Driver) DumpEvents() ([]schema.Event, error) {
	var events []schema.Event
	client := d.newClient()
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

	response, err := client.DescribeSASSuspEvents(context.Background(), api.DefaultRegion)
	if err != nil {
		return events, err
	}
	for _, event := range response.SuspEvents {
		_event := schema.Event{
			Id:       event.SecurityEventIDs,
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
	client := d.newClient()
	ids := strings.Split(eid, ",")
	cleaned := make([]string, 0, len(ids))
	for _, id := range ids {
		if v := strings.TrimSpace(id); v != "" {
			cleaned = append(cleaned, v)
		}
	}
	response, err := client.HandleSASSecurityEvents(context.Background(), api.DefaultRegion, "advance_mark_mis_info", cleaned)
	if err != nil {
		logger.Error(err)
		return
	}
	content, err := json.Marshal(response)
	if err != nil {
		logger.Error(err)
		return
	}
	fmt.Println(string(content))
}
