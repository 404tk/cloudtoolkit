package api

// Cloud Logging — entries.list endpoint used by event-check.

const LoggingBaseURL = "https://logging.googleapis.com"

// LogEntry maps the subset of a Cloud Logging LogEntry resource that
// event-check surfaces. Real responses include many more fields (json /
// proto payloads, severity, labels) we don't need.
type LogEntry struct {
	InsertID         string                 `json:"insertId"`
	LogName          string                 `json:"logName"`
	Timestamp        string                 `json:"timestamp"`
	ReceiveTimestamp string                 `json:"receiveTimestamp"`
	Severity         string                 `json:"severity"`
	Resource         LogEntryResource       `json:"resource"`
	ProtoPayload     LogProtoPayload        `json:"protoPayload"`
	Operation        LogEntryOperation      `json:"operation"`
	Labels           map[string]string      `json:"labels"`
}

type LogEntryResource struct {
	Type   string            `json:"type"`
	Labels map[string]string `json:"labels"`
}

type LogProtoPayload struct {
	Type           string                 `json:"@type"`
	ServiceName    string                 `json:"serviceName"`
	MethodName     string                 `json:"methodName"`
	ResourceName   string                 `json:"resourceName"`
	AuthInfo       LogProtoAuthInfo       `json:"authenticationInfo"`
	RequestMeta    LogProtoRequestMeta    `json:"requestMetadata"`
	Status         LogProtoStatus         `json:"status"`
	AuthorizationInfo []LogProtoAuthorize `json:"authorizationInfo"`
}

type LogProtoAuthInfo struct {
	PrincipalEmail string `json:"principalEmail"`
}

type LogProtoRequestMeta struct {
	CallerIP                string `json:"callerIp"`
	CallerSuppliedUserAgent string `json:"callerSuppliedUserAgent"`
}

type LogProtoStatus struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type LogProtoAuthorize struct {
	Resource   string `json:"resource"`
	Permission string `json:"permission"`
	Granted    bool   `json:"granted"`
}

type LogEntryOperation struct {
	ID       string `json:"id"`
	Producer string `json:"producer"`
	First    bool   `json:"first"`
	Last     bool   `json:"last"`
}

type ListLogEntriesRequest struct {
	ResourceNames []string `json:"resourceNames"`
	Filter        string   `json:"filter,omitempty"`
	OrderBy       string   `json:"orderBy,omitempty"`
	PageSize      int      `json:"pageSize,omitempty"`
	PageToken     string   `json:"pageToken,omitempty"`
}

type ListLogEntriesResponse struct {
	Entries       []LogEntry `json:"entries"`
	NextPageToken string     `json:"nextPageToken"`
}

// ListLogsResponse is the typed result of `projects/<p>/logs.list`. Returns
// just the log names available in the project — one per cloudlist `log`
// asset entry, which is a closer fit than the heavy `entries.list` payload.
type ListLogsResponse struct {
	LogNames      []string `json:"logNames"`
	NextPageToken string   `json:"nextPageToken"`
}
