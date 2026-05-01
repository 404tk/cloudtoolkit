package replay

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

type ssmInvocation struct {
	CommandID  string
	InstanceID string
	Output     string
}

func (t *transport) handleSSM(req *http.Request, body []byte) (*http.Response, error) {
	if req.Method != http.MethodPost {
		return ssmJSONError(req, http.StatusMethodNotAllowed, "InvalidAction", "ssm replay expects POST"), nil
	}
	target := strings.TrimSpace(req.Header.Get("X-Amz-Target"))
	switch target {
	case "AmazonSSM.SendCommand":
		return t.handleSSMSendCommand(req, body)
	case "AmazonSSM.GetCommandInvocation":
		return t.handleSSMGetCommandInvocation(req, body)
	}
	return ssmJSONError(req, http.StatusBadRequest, "InvalidAction", fmt.Sprintf("unsupported ssm target: %s", target)), nil
}

func (t *transport) handleSSMSendCommand(req *http.Request, body []byte) (*http.Response, error) {
	var input api.SendCommandInput
	if err := json.Unmarshal(body, &input); err != nil {
		return ssmJSONError(req, http.StatusBadRequest, "ValidationException", err.Error()), nil
	}
	if len(input.InstanceIDs) == 0 {
		return ssmJSONError(req, http.StatusBadRequest, "ValidationException", "InstanceIds is required"), nil
	}
	commands := input.Parameters["commands"]
	command := strings.Join(commands, "\n")

	t.mu.Lock()
	t.sequence++
	commandID := fmt.Sprintf("ssm-cmd-%05d", t.sequence)
	if t.ssmInvocations == nil {
		t.ssmInvocations = make(map[string]ssmInvocation)
	}
	for _, instance := range input.InstanceIDs {
		key := commandID + ":" + instance
		t.ssmInvocations[key] = ssmInvocation{
			CommandID:  commandID,
			InstanceID: instance,
			Output:     ssmReplayOutput(instance, command),
		}
	}
	t.mu.Unlock()

	resp := api.SendCommandOutput{}
	resp.Command.CommandID = commandID
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func (t *transport) handleSSMGetCommandInvocation(req *http.Request, body []byte) (*http.Response, error) {
	var input api.GetCommandInvocationInput
	if err := json.Unmarshal(body, &input); err != nil {
		return ssmJSONError(req, http.StatusBadRequest, "ValidationException", err.Error()), nil
	}
	key := strings.TrimSpace(input.CommandID) + ":" + strings.TrimSpace(input.InstanceID)
	t.mu.Lock()
	invocation, ok := t.ssmInvocations[key]
	t.mu.Unlock()
	if !ok {
		return ssmJSONError(req, http.StatusNotFound, "InvocationDoesNotExist", "Command did not run on the specified instance"), nil
	}
	resp := api.GetCommandInvocationOutput{
		CommandID:             invocation.CommandID,
		InstanceID:            invocation.InstanceID,
		Status:                "Success",
		StandardOutputContent: invocation.Output,
	}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

// ssmReplayOutput returns a deterministic, recognisably-fake stdout for a
// (instance, command) pair. The validation flow is more interested in *that*
// the command landed than in any specific output, so a stable canned reply
// is enough — and mirroring the (instance,command)-keyed lookup used by the
// alibaba replay keeps cross-provider behavior consistent.
func ssmReplayOutput(instance, command string) string {
	h := sha1.New()
	h.Write([]byte(instance))
	h.Write([]byte{':'})
	h.Write([]byte(command))
	digest := hex.EncodeToString(h.Sum(nil))[:12]
	switch strings.TrimSpace(command) {
	case "":
		return ""
	case "whoami":
		return "ec2-user"
	case "id":
		return "uid=1000(ec2-user) gid=1000(ec2-user) groups=1000(ec2-user)"
	case "pwd":
		return "/home/ec2-user"
	}
	return fmt.Sprintf("ssm-replay output for %s (%s)\n", instance, digest)
}

func ssmJSONError(req *http.Request, statusCode int, code, message string) *http.Response {
	envelope := map[string]string{
		"__type":  code,
		"Message": message,
	}
	return demoreplay.JSONResponse(req, statusCode, envelope)
}
