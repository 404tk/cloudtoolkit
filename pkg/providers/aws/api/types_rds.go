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
