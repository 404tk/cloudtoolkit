package api

const NetworkAPIVersion = "2022-07-01"

type NetworkInterface struct {
	ID         string                `json:"id"`
	Name       string                `json:"name"`
	Properties NetworkInterfaceProps `json:"properties"`
}

type NetworkInterfaceProps struct {
	IPConfigurations []IPConfiguration `json:"ipConfigurations"`
}

type IPConfiguration struct {
	Name       string               `json:"name"`
	Properties IPConfigurationProps `json:"properties"`
}

type IPConfigurationProps struct {
	PrivateIPAddress string       `json:"privateIPAddress"`
	PublicIPAddress  *ResourceRef `json:"publicIPAddress,omitempty"`
}

type ResourceRef struct {
	ID string `json:"id"`
}

type PublicIPAddress struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	Properties PublicIPAddressProps `json:"properties"`
}

type PublicIPAddressProps struct {
	IPAddress string `json:"ipAddress"`
}
