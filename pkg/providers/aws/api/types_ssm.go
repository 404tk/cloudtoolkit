package api

import (
	"context"
	"encoding/json"
	"net/http"
)

// AWS SSM API constants — JSON-1.1 RPC endpoint with X-Amz-Target dispatch.
const (
	ssmContentType = "application/x-amz-json-1.1"
	ssmTargetSend  = "AmazonSSM.SendCommand"
	ssmTargetGet   = "AmazonSSM.GetCommandInvocation"

	// SSMDocumentLinux / SSMDocumentWindows are the canonical AWS-managed SSM
	// documents the validation flow uses to run shell or PowerShell commands.
	SSMDocumentLinux   = "AWS-RunShellScript"
	SSMDocumentWindows = "AWS-RunPowerShellScript"
)

type SendCommandInput struct {
	DocumentName string              `json:"DocumentName"`
	InstanceIDs  []string            `json:"InstanceIds"`
	Parameters   map[string][]string `json:"Parameters"`
}

type SendCommandOutput struct {
	Command struct {
		CommandID string `json:"CommandId"`
	} `json:"Command"`
}

type GetCommandInvocationInput struct {
	CommandID  string `json:"CommandId"`
	InstanceID string `json:"InstanceId"`
}

type GetCommandInvocationOutput struct {
	CommandID             string `json:"CommandId"`
	InstanceID            string `json:"InstanceId"`
	Status                string `json:"Status"`
	StatusDetails         string `json:"StatusDetails"`
	StandardOutputContent string `json:"StandardOutputContent"`
	StandardErrorContent  string `json:"StandardErrorContent"`
}

// SSMSendCommand kicks off a command execution against one or more instances.
// `documentName` is `AWS-RunShellScript` (linux) or `AWS-RunPowerShellScript`
// (windows). `commands` is the list of shell lines to execute.
func (c *Client) SSMSendCommand(ctx context.Context, region, documentName string, instanceIDs, commands []string) (SendCommandOutput, error) {
	body, err := json.Marshal(SendCommandInput{
		DocumentName: documentName,
		InstanceIDs:  instanceIDs,
		Parameters:   map[string][]string{"commands": commands},
	})
	if err != nil {
		return SendCommandOutput{}, err
	}
	headers := http.Header{}
	headers.Set("Content-Type", ssmContentType)
	headers.Set("X-Amz-Target", ssmTargetSend)
	var out SendCommandOutput
	err = c.DoRESTJSON(ctx, Request{
		Service: "ssm",
		Region:  region,
		Method:  http.MethodPost,
		Path:    "/",
		Body:    body,
		Headers: headers,
	}, &out)
	return out, err
}

// SSMGetCommandInvocation polls SSM for the result of a previously-sent
// command on a single instance.
func (c *Client) SSMGetCommandInvocation(ctx context.Context, region, commandID, instanceID string) (GetCommandInvocationOutput, error) {
	body, err := json.Marshal(GetCommandInvocationInput{
		CommandID:  commandID,
		InstanceID: instanceID,
	})
	if err != nil {
		return GetCommandInvocationOutput{}, err
	}
	headers := http.Header{}
	headers.Set("Content-Type", ssmContentType)
	headers.Set("X-Amz-Target", ssmTargetGet)
	var out GetCommandInvocationOutput
	err = c.DoRESTJSON(ctx, Request{
		Service:    "ssm",
		Region:     region,
		Method:     http.MethodPost,
		Path:       "/",
		Body:       body,
		Headers:    headers,
		Idempotent: true,
	}, &out)
	return out, err
}
