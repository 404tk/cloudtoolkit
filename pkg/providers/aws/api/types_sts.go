package api

import (
	"context"
	"encoding/xml"
	"net/http"
)

type GetCallerIdentityOutput struct {
	Account   string
	Arn       string
	UserID    string
	RequestID string
}

type getCallerIdentityResponse struct {
	XMLName                 xml.Name                      `xml:"GetCallerIdentityResponse"`
	GetCallerIdentityResult getCallerIdentityResult       `xml:"GetCallerIdentityResult"`
	ResponseMetadata        getCallerIdentityResponseMeta `xml:"ResponseMetadata"`
}

type getCallerIdentityResult struct {
	Account string `xml:"Account"`
	Arn     string `xml:"Arn"`
	UserID  string `xml:"UserId"`
}

type getCallerIdentityResponseMeta struct {
	RequestID string `xml:"RequestId"`
}

func (c *Client) GetCallerIdentity(ctx context.Context, region string) (GetCallerIdentityOutput, error) {
	region = normalizeSTSRegion(region)
	var wire getCallerIdentityResponse
	err := c.DoXML(ctx, Request{
		Service:    "sts",
		Region:     region,
		Action:     "GetCallerIdentity",
		Version:    "2011-06-15",
		Method:     http.MethodPost,
		Path:       "/",
		Idempotent: true,
	}, &wire)
	if err != nil {
		return GetCallerIdentityOutput{}, err
	}
	return GetCallerIdentityOutput{
		Account:   wire.GetCallerIdentityResult.Account,
		Arn:       wire.GetCallerIdentityResult.Arn,
		UserID:    wire.GetCallerIdentityResult.UserID,
		RequestID: wire.ResponseMetadata.RequestID,
	}, nil
}

func normalizeSTSRegion(region string) string {
	return normalizeRegion(region)
}
