package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/xml"
	"net/http"
	"net/url"
	"strings"
)

const rdsAPIVersion = "2014-10-31"

type ModifyDBInstanceOutput struct {
	DBInstanceIdentifier string
	DBInstanceStatus     string
	MasterUsername       string
	RequestID            string
}

type modifyDBInstanceResponse struct {
	XMLName                xml.Name                  `xml:"ModifyDBInstanceResponse"`
	ModifyDBInstanceResult modifyDBInstanceResultWire `xml:"ModifyDBInstanceResult"`
	Metadata               rdsResponseMetadata        `xml:"ResponseMetadata"`
}

type modifyDBInstanceResultWire struct {
	DBInstance dbInstanceWire `xml:"DBInstance"`
}

type dbInstanceWire struct {
	DBInstanceIdentifier string `xml:"DBInstanceIdentifier"`
	DBInstanceStatus     string `xml:"DBInstanceStatus"`
	MasterUsername       string `xml:"MasterUsername"`
}

type rdsResponseMetadata struct {
	RequestID string `xml:"RequestId"`
}

// ModifyDBInstanceMasterPassword rotates the master user password on an RDS
// instance. AWS RDS has no notion of "create user via API" — accounts live in
// the database engine itself; the CSPM-relevant management-plane signal is
// the master password rotation captured in CloudTrail.
func (c *Client) ModifyDBInstanceMasterPassword(ctx context.Context, region, instanceID, masterPassword string) (ModifyDBInstanceOutput, error) {
	query := url.Values{}
	query.Set("DBInstanceIdentifier", strings.TrimSpace(instanceID))
	query.Set("MasterUserPassword", masterPassword)
	query.Set("ApplyImmediately", "true")
	var wire modifyDBInstanceResponse
	if err := c.DoXML(ctx, Request{
		Service: "rds",
		Region:  region,
		Action:  "ModifyDBInstance",
		Version: rdsAPIVersion,
		Method:  http.MethodPost,
		Path:    "/",
		Query:   query,
	}, &wire); err != nil {
		return ModifyDBInstanceOutput{}, err
	}
	return ModifyDBInstanceOutput{
		DBInstanceIdentifier: strings.TrimSpace(wire.ModifyDBInstanceResult.DBInstance.DBInstanceIdentifier),
		DBInstanceStatus:     strings.TrimSpace(wire.ModifyDBInstanceResult.DBInstance.DBInstanceStatus),
		MasterUsername:       strings.TrimSpace(wire.ModifyDBInstanceResult.DBInstance.MasterUsername),
		RequestID:            strings.TrimSpace(wire.Metadata.RequestID),
	}, nil
}

// RandomPassword returns a base64 string suitable for an AWS RDS master
// password rotation when callers want to lock out access (the `userdel`
// branch). 24 random bytes encode to 32 base64 chars — well below the RDS
// 41-char limit and above the 8-char minimum.
func RandomPassword() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return strings.ReplaceAll(base64.StdEncoding.EncodeToString(buf), "/", "_"), nil
}

// DBInstance is the typed AWS RDS instance shape returned by
// DescribeDBInstances. Only the fields cloudlist surfaces are projected.
type DBInstance struct {
	DBInstanceIdentifier string
	Engine               string
	EngineVersion        string
	DBName               string
	Status               string
	PubliclyAccessible   bool
	Address              string
	Port                 int64
	AvailabilityZone     string
}

type DescribeDBInstancesOutput struct {
	DBInstances []DBInstance
	Marker      string
	RequestID   string
}

type describeDBInstancesResponse struct {
	XMLName                   xml.Name                       `xml:"DescribeDBInstancesResponse"`
	DescribeDBInstancesResult describeDBInstancesResultWire  `xml:"DescribeDBInstancesResult"`
	Metadata                  rdsResponseMetadata            `xml:"ResponseMetadata"`
}

type describeDBInstancesResultWire struct {
	DBInstances []dbInstanceDescribeWire `xml:"DBInstances>DBInstance"`
	Marker      string                   `xml:"Marker"`
}

type dbInstanceDescribeWire struct {
	DBInstanceIdentifier string                 `xml:"DBInstanceIdentifier"`
	Engine               string                 `xml:"Engine"`
	EngineVersion        string                 `xml:"EngineVersion"`
	DBName               string                 `xml:"DBName"`
	DBInstanceStatus     string                 `xml:"DBInstanceStatus"`
	PubliclyAccessible   bool                   `xml:"PubliclyAccessible"`
	Endpoint             dbInstanceEndpointWire `xml:"Endpoint"`
	AvailabilityZone     string                 `xml:"AvailabilityZone"`
}

type dbInstanceEndpointWire struct {
	Address string `xml:"Address"`
	Port    int64  `xml:"Port"`
}

// DescribeDBInstances paginates through RDS DescribeDBInstances. Pass an
// empty marker for the first call.
func (c *Client) DescribeDBInstances(ctx context.Context, region, marker string) (DescribeDBInstancesOutput, error) {
	query := url.Values{}
	if marker = strings.TrimSpace(marker); marker != "" {
		query.Set("Marker", marker)
	}
	var wire describeDBInstancesResponse
	err := c.DoXML(ctx, Request{
		Service:    "rds",
		Region:     region,
		Action:     "DescribeDBInstances",
		Version:    rdsAPIVersion,
		Method:     http.MethodPost,
		Path:       "/",
		Query:      query,
		Idempotent: true,
	}, &wire)
	if err != nil {
		return DescribeDBInstancesOutput{}, err
	}
	out := DescribeDBInstancesOutput{
		DBInstances: make([]DBInstance, 0, len(wire.DescribeDBInstancesResult.DBInstances)),
		Marker:      strings.TrimSpace(wire.DescribeDBInstancesResult.Marker),
		RequestID:   strings.TrimSpace(wire.Metadata.RequestID),
	}
	for _, w := range wire.DescribeDBInstancesResult.DBInstances {
		out.DBInstances = append(out.DBInstances, DBInstance{
			DBInstanceIdentifier: strings.TrimSpace(w.DBInstanceIdentifier),
			Engine:               strings.TrimSpace(w.Engine),
			EngineVersion:        strings.TrimSpace(w.EngineVersion),
			DBName:               strings.TrimSpace(w.DBName),
			Status:               strings.TrimSpace(w.DBInstanceStatus),
			PubliclyAccessible:   w.PubliclyAccessible,
			Address:              strings.TrimSpace(w.Endpoint.Address),
			Port:                 w.Endpoint.Port,
			AvailabilityZone:     strings.TrimSpace(w.AvailabilityZone),
		})
	}
	return out, nil
}
