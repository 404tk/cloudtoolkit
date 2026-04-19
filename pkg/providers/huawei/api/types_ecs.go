package api

type ListECSServersDetailsResponse struct {
	Count   int32             `json:"count"`
	Servers []ECSServerDetail `json:"servers"`
}

type ECSServerDetail struct {
	Status    string                        `json:"status"`
	Name      string                        `json:"name"`
	Addresses map[string][]ECSServerAddress `json:"addresses"`
}

type ECSServerAddress struct {
	Addr         string `json:"addr"`
	OSEXTIPStype string `json:"OS-EXT-IPS:type"`
}
