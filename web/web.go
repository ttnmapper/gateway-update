package web

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"ttnmapper-gateway-update/types"
)

var Url = "https://www.thethingsnetwork.org/gateway-data/"

func FetchWebStatuses() (map[string]*types.WebGateway, error) {
	log.Println("Fetching web statuses")

	httpClient := http.Client{
		Timeout: time.Second * 60, // Maximum of 1 minute
	}

	req, err := http.NewRequest(http.MethodGet, Url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "ttnmapper-update-gateway")

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	webData := map[string]*types.WebGateway{}
	err = json.NewDecoder(res.Body).Decode(&webData)
	if err != nil {
		return nil, err
	}
	return webData, nil
}

func WebGatewayToTtnMapperGateway(gatewayIn types.WebGateway) types.TtnMapperGateway {
	gatewayOut := types.TtnMapperGateway{}

	// Website lists only TTN gateways
	gatewayOut.NetworkId = "thethingsnetwork.org"

	gatewayOut.GatewayId = gatewayIn.ID

	if gatewayIn.LastSeen != nil {
		gatewayOut.Time = gatewayIn.LastSeen.UnixNano()
	}

	// eui-c0ee40ffff29618d
	if len(gatewayIn.ID) == 20 && strings.HasPrefix(gatewayIn.ID, "eui-") {
		gatewayOut.GatewayEui = strings.ToUpper(strings.TrimPrefix(gatewayIn.ID, "eui-"))
	}
	// 00800000a000222e
	_, err := strconv.ParseUint(gatewayIn.ID, 16, 64)
	if err == nil {
		// Is a valid hex number
		if len(gatewayIn.ID) == 16 {
			gatewayOut.GatewayEui = strings.ToUpper(gatewayIn.ID)
		}
	}

	// Only use location obtained from web api
	gatewayOut.Latitude = gatewayIn.Location.Latitude
	gatewayOut.Longitude = gatewayIn.Location.Longitude
	gatewayOut.Altitude = int32(gatewayIn.Location.Altitude)

	gatewayOut.Description = gatewayIn.Description

	return gatewayOut
}
