package types

import "time"

type NocStatus struct {
	Statuses map[string]NocGateway `json:"statuses"`
}

type NocGateway struct {
	/*
		{
			"statuses":
				{
					"00-08-00-4a-0b-34":
					{
						"timestamp":"2020-05-08T15:48:05.264780623Z",
						"authenticated":true,
						"uplink":"411386",
						"downlink":"73571",
						"location":
						{
							"latitude":46.67002,
							"longitude":0.3634259,
							"source":"REGISTRY"
						},
						"frequency_plan":"EU_863_870",
						"platform":"MultiTech",
						"gps":
						{
							"latitude":46.67002,
							"longitude":0.3634259,
							"source":"REGISTRY"
						},
						"time":"1588952885264780623",
						"rx_ok":411386,
						"tx_in":73571
					},
	*/
	Timestamp     time.Time `json:"timestamp,omitempty"`
	Authenticated bool      `json:"authenticated,omitempty"`
	Uplink        string    `json:"uplink,omitempty"`
	Downlink      string    `json:"downlink,omitempty"`
	Location      struct {
		Latitude  float64 `json:"latitude,omitempty"`
		Longitude float64 `json:"longitude,omitempty"`
		Source    string  `json:"source,omitempty"`
	} `json:"location"`
	FrequencyPlan string `json:"frequency_plan,omitempty"`
	Platform      string `json:"platform,omitempty"`
	Gps           struct {
		Latitude  float64 `json:"latitude,omitempty"`
		Longitude float64 `json:"longitude,omitempty"`
		Source    string  `json:"source,omitempty"`
	} `json:"gps"`
	Time string `json:"time,omitempty"`
	RxOk int    `json:"rx_ok,omitempty"`
	TxIn int    `json:"tx_in,omitempty"`
}

// This is not final. This version obtained from @htdvisser
type WebGateway struct {
	ID          string   `json:"id"`
	Description string   `json:"description,omitempty"`
	Owner       string   `json:"owner,omitempty"`
	Owners      []string `json:"owners,omitempty"`
	Location    struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Altitude  int64   `json:"altitude"`
	} `json:"location,omitempty"`
	CountryCode string            `json:"country_code,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
	LastSeen    *time.Time        `json:"last_seen,omitempty"`
}
