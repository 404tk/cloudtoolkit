package api

// Huawei LTS (Log Tank Service) ListLogGroups — `GET /v2/{project_id}/groups`
// returns the LTS log groups inside a project, mirroring the AWS CloudWatch
// Logs surface. The cloudlist `log` asset surfaces one row per log group.

type LTSLogGroup struct {
	LogGroupID         string `json:"log_group_id"`
	LogGroupName       string `json:"log_group_name"`
	CreationTime       int64  `json:"creation_time"`
	TTLInDays          int64  `json:"ttl_in_days"`
	LogStreamNameAlias string `json:"log_stream_name_alias"`
}

type ListLogGroupsResponse struct {
	LogGroups []LTSLogGroup `json:"log_groups"`
}
