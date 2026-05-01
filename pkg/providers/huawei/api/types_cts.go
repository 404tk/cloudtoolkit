package api

type ListTracesResponse struct {
	Traces   []Trace       `json:"traces"`
	MetaData TraceMetaData `json:"meta_data"`
}

type TraceMetaData struct {
	Count  int    `json:"count"`
	Marker string `json:"marker"`
}

type Trace struct {
	TraceID      string    `json:"trace_id"`
	TraceName    string    `json:"trace_name"`
	TraceRating  string    `json:"trace_rating"`
	TraceType    string    `json:"trace_type"`
	Code         string    `json:"code"`
	APIService   string    `json:"service_type"`
	OperationID  string    `json:"operation_id"`
	ResourceID   string    `json:"resource_id"`
	ResourceName string    `json:"resource_name"`
	ResourceType string    `json:"resource_type"`
	SourceIP     string    `json:"source_ip"`
	Time         int64     `json:"time"`
	User         TraceUser `json:"user"`
}

type TraceUser struct {
	AccessKeyID string `json:"access_key_id"`
	UserName    string `json:"user_name"`
	Name        string `json:"name"`
}
