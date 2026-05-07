package replay

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

type assistantInvocationFixture struct {
	CommandID  string
	InstanceID string
}

func (t *transport) handleAssistant(req *http.Request, body []byte) (*http.Response, error) {
	if req.Method != http.MethodPost {
		return apiErrorResponse(req, http.StatusMethodNotAllowed, "MethodNotAllowed",
			"Cloud Assistant replay expects POST requests"), nil
	}

	path := req.URL.Path
	switch {
	case strings.HasSuffix(path, "/createCommand"):
		var payload api.CreateCommandRequest
		if err := json.Unmarshal(body, &payload); err != nil {
			return apiErrorResponse(req, http.StatusBadRequest, "InvalidParameter", err.Error()), nil
		}
		if strings.TrimSpace(payload.CommandContent) == "" {
			return apiErrorResponse(req, http.StatusBadRequest, "InvalidParameter", "commandContent is required"), nil
		}
		commandID := t.addAssistantCommand(payload)
		resp := api.CreateCommandResponse{RequestID: "req-replay-assistant-create-command"}
		resp.Result.CommandID = commandID
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case strings.HasSuffix(path, "/invokeCommand"):
		var payload api.InvokeCommandRequest
		if err := json.Unmarshal(body, &payload); err != nil {
			return apiErrorResponse(req, http.StatusBadRequest, "InvalidParameter", err.Error()), nil
		}
		commandID := strings.TrimSpace(payload.CommandID)
		if commandID == "" || len(payload.Instances) == 0 || strings.TrimSpace(payload.Instances[0]) == "" {
			return apiErrorResponse(req, http.StatusBadRequest, "InvalidParameter", "commandId and instances are required"), nil
		}
		if _, ok := t.snapshotAssistantCommand(commandID); !ok {
			return apiErrorResponse(req, http.StatusNotFound, "ResourceNotFound", "command not found"), nil
		}
		invokeID := t.addAssistantInvocation(commandID, payload.Instances[0])
		resp := api.InvokeCommandResponse{RequestID: "req-replay-assistant-invoke-command"}
		resp.Result.InvokeID = invokeID
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case strings.HasSuffix(path, "/describeInvocations"):
		var payload api.DescribeInvocationsRequest
		if err := json.Unmarshal(body, &payload); err != nil {
			return apiErrorResponse(req, http.StatusBadRequest, "InvalidParameter", err.Error()), nil
		}
		if len(payload.InvokeIDs) == 0 || strings.TrimSpace(payload.InvokeIDs[0]) == "" {
			return apiErrorResponse(req, http.StatusBadRequest, "InvalidParameter", "invokeIds is required"), nil
		}
		invocation, command, ok := t.snapshotAssistantInvocation(payload.InvokeIDs[0])
		if !ok {
			return apiErrorResponse(req, http.StatusNotFound, "ResourceNotFound", "invocation not found"), nil
		}
		resp := api.DescribeInvocationsResponse{RequestID: "req-replay-assistant-describe-invocations"}
		resp.Result.TotalCount = 1
		resp.Result.Invocations = []api.Invocation{assistantInvocationResponse(payload.InvokeIDs[0], invocation, command)}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case strings.HasSuffix(path, "/deleteCommands"):
		var payload api.DeleteCommandsRequest
		if err := json.Unmarshal(body, &payload); err != nil {
			return apiErrorResponse(req, http.StatusBadRequest, "InvalidParameter", err.Error()), nil
		}
		t.deleteAssistantCommands(payload.CommandIDs)
		resp := api.DeleteCommandsResponse{RequestID: "req-replay-assistant-delete-commands"}
		if len(payload.CommandIDs) > 0 {
			resp.Result.CommandID = strings.TrimSpace(payload.CommandIDs[0])
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	}
	return apiErrorResponse(req, http.StatusNotFound, "InvalidPath",
		"unsupported assistant path: "+path), nil
}

func (t *transport) addAssistantCommand(payload api.CreateCommandRequest) string {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.assistantSeq++
	commandID := fmt.Sprintf("cmd-replay-assistant-%03d", t.assistantSeq)
	t.assistantCommands[commandID] = payload
	return commandID
}

func (t *transport) snapshotAssistantCommand(commandID string) (api.CreateCommandRequest, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	command, ok := t.assistantCommands[commandID]
	return command, ok
}

func (t *transport) addAssistantInvocation(commandID, instanceID string) string {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.assistantSeq++
	invokeID := fmt.Sprintf("inv-replay-assistant-%03d", t.assistantSeq)
	t.assistantInvocations[invokeID] = assistantInvocationFixture{
		CommandID:  commandID,
		InstanceID: strings.TrimSpace(instanceID),
	}
	return invokeID
}

func (t *transport) snapshotAssistantInvocation(invokeID string) (assistantInvocationFixture, api.CreateCommandRequest, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	invocation, ok := t.assistantInvocations[strings.TrimSpace(invokeID)]
	if !ok {
		return assistantInvocationFixture{}, api.CreateCommandRequest{}, false
	}
	command, ok := t.assistantCommands[invocation.CommandID]
	if !ok {
		return assistantInvocationFixture{}, api.CreateCommandRequest{}, false
	}
	return invocation, command, true
}

func (t *transport) deleteAssistantCommands(commandIDs []string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, commandID := range commandIDs {
		delete(t.assistantCommands, strings.TrimSpace(commandID))
	}
}

func assistantInvocationResponse(invokeID string, invocation assistantInvocationFixture, command api.CreateCommandRequest) api.Invocation {
	output := assistantReplayOutput(command.CommandContent)
	return api.Invocation{
		Status:      "finish",
		CommandID:   invocation.CommandID,
		InvokeID:    strings.TrimSpace(invokeID),
		CommandType: command.CommandType,
		InvocationInstances: []api.InvocationInstance{
			{
				InstanceID: invocation.InstanceID,
				Status:     "finish",
				ExitCode:   "0",
				Output:     base64.StdEncoding.EncodeToString([]byte(output)),
			},
		},
	}
}

func assistantReplayOutput(contentB64 string) string {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(contentB64))
	if err != nil || len(raw) == 0 {
		return "jdcloud replay command completed\n"
	}
	return "jdcloud replay accepted command: " + string(raw) + "\n"
}
