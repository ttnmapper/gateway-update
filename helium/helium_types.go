package helium

import "time"

type HotspotApiResponse struct {
	Data   []Hotspot `json:"data"`
	Cursor string    `json:"cursor"`
}

/*
{
   "lng":-78.74710984473072,
   "lat":42.99533035889439,
   "timestamp_added":"2021-11-13T17:29:11.000000Z",
   "status":{
      "timestamp":null,
      "online":"online",
      "listen_addrs":null,
      "height":null
   },
   "reward_scale":null,
   "payer":"13ENbEQPAvytjLnqavnbSAzurhGoCSNkGECMx7eHHDAfEaDirdY",
   "owner":"135SUwbieAFXYXEbZbNZdHW8U9kAkyRCAguojRp3Sv2reWjsyhd",
   "nonce":1,
   "name":"bubbly-honeysuckle-dolphin",
   "mode":"full",
   "location_hex":"882aa6d823fffff",
   "location":"8c2aa6d823927ff",
   "last_poc_challenge":null,
   "last_change_block":1097250,
   "geocode":{
      "short_street":"Presidents Walk",
      "short_state":"NY",
      "short_country":"US",
      "short_city":"Buffalo",
      "long_street":"Presidents Walk",
      "long_state":"New York",
      "long_country":"United States",
      "long_city":"Buffalo",
      "city_id":"YnVmZmFsb25ldyB5b3JrdW5pdGVkIHN0YXRlcw"
   },
   "gain":90,
   "elevation":5,
   "block_added":1097250,
   "block":1097251,
   "address":"11j4jiWSv44uJG351X7dXyXvrfPNoio68Jggo7Qu2RCeNNS4rET"
}
*/
type Hotspot struct {
	Longitude        float64       `json:"lng"`
	Latitude         float64       `json:"lat"`
	TimestampAdded   time.Time     `json:"timestamp_added"`
	Status           HotspotStatus `json:"status"`
	RewardScale      interface{}   `json:"reward_scale"`
	Payer            string        `json:"payer"`
	Owner            string        `json:"owner"`
	Nonce            int           `json:"nonce"`
	Name             string        `json:"name"`
	Mode             string        `json:"mode"`
	LocationHex      string        `json:"location_hex"`
	Location         string        `json:"location"`
	LastPocChallenge interface{}   `json:"last_poc_challenge"`
	LastChangeBlock  int           `json:"last_change_block"`
	Geocode          interface{}   `json:"geocode"`
	Gain             int           `json:"gain"`
	Elevation        int32         `json:"elevation"`
	BlockAdded       int           `json:"block_added"`
	Block            int           `json:"block"`
	Address          string        `json:"address"`
}

/*
   "status":{
      "timestamp":null,
      "online":"online",
      "listen_addrs":null,
      "height":null
   },
*/
type HotspotStatus struct {
	Timestamp   time.Time   `json:"timestamp"`
	Online      string      `json:"online"`
	ListenAddrs interface{} `json:"listen_addrs"`
	Height      interface{} `json:"height"`
}
