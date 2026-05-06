package replay

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"sync"

	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

var (
	rdsMu             sync.Mutex
	rdsMasterPassword = make(map[string]string)
)

func (t *transport) handleRDS(req *http.Request, body []byte) (*http.Response, error) {
	form, err := parseFormBody(body)
	if err != nil {
		return apiErrorResponse(req, http.StatusBadRequest, "MalformedQueryString", err.Error()), nil
	}
	action := form.Get("Action")
	switch action {
	case "DescribeDBInstances":
		region := regionFromHost(req.URL.Hostname())
		instances := rdsInstancesForRegion(region)
		envelope := describeDBInstancesReplyEnvelope{
			Result: describeDBInstancesReplyResult{
				DBInstances: instances,
			},
			Metadata: rdsResponseMeta{RequestID: "req-replay-rds-describe-db-instances"},
		}
		return demoreplay.XMLResponse(req, http.StatusOK, envelope), nil
	case "ModifyDBInstance":
		instance := strings.TrimSpace(form.Get("DBInstanceIdentifier"))
		password := form.Get("MasterUserPassword")
		if instance == "" || password == "" {
			return apiErrorResponse(req, http.StatusBadRequest, "ValidationError",
				"DBInstanceIdentifier and MasterUserPassword required"), nil
		}
		rdsMu.Lock()
		rdsMasterPassword[instance] = password
		rdsMu.Unlock()
		return demoreplay.XMLResponse(req, http.StatusOK, modifyDBInstanceReplyEnvelope{
			Result: modifyDBInstanceReplyResult{
				DBInstance: dbInstanceReply{
					DBInstanceIdentifier: instance,
					DBInstanceStatus:     "modifying",
					MasterUsername:       "admin",
				},
			},
			Metadata: rdsResponseMeta{RequestID: "req-replay-rds-modify-db-instance"},
		}), nil
	}
	return apiErrorResponse(req, http.StatusBadRequest, "InvalidAction",
		fmt.Sprintf("unsupported rds action: %s", action)), nil
}

type modifyDBInstanceReplyEnvelope struct {
	XMLName  xml.Name                    `xml:"ModifyDBInstanceResponse"`
	Result   modifyDBInstanceReplyResult `xml:"ModifyDBInstanceResult"`
	Metadata rdsResponseMeta             `xml:"ResponseMetadata"`
}

type modifyDBInstanceReplyResult struct {
	DBInstance dbInstanceReply `xml:"DBInstance"`
}

type dbInstanceReply struct {
	DBInstanceIdentifier string `xml:"DBInstanceIdentifier"`
	DBInstanceStatus     string `xml:"DBInstanceStatus"`
	MasterUsername       string `xml:"MasterUsername"`
}

type rdsResponseMeta struct {
	RequestID string `xml:"RequestId"`
}

type describeDBInstancesReplyEnvelope struct {
	XMLName  xml.Name                       `xml:"DescribeDBInstancesResponse"`
	Result   describeDBInstancesReplyResult `xml:"DescribeDBInstancesResult"`
	Metadata rdsResponseMeta                `xml:"ResponseMetadata"`
}

type describeDBInstancesReplyResult struct {
	DBInstances []describeDBInstanceWire `xml:"DBInstances>DBInstance"`
	Marker      string                   `xml:"Marker,omitempty"`
}

type describeDBInstanceWire struct {
	DBInstanceIdentifier string                         `xml:"DBInstanceIdentifier"`
	Engine               string                         `xml:"Engine"`
	EngineVersion        string                         `xml:"EngineVersion"`
	DBName               string                         `xml:"DBName"`
	DBInstanceStatus     string                         `xml:"DBInstanceStatus"`
	PubliclyAccessible   bool                           `xml:"PubliclyAccessible"`
	Endpoint             describeDBInstanceEndpointWire `xml:"Endpoint"`
	AvailabilityZone     string                         `xml:"AvailabilityZone"`
}

type describeDBInstanceEndpointWire struct {
	Address string `xml:"Address"`
	Port    int64  `xml:"Port"`
}
