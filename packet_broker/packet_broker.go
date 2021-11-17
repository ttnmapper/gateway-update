package packet_broker

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"
	"ttnmapper-gateway-update/packet_broker/Openapi"
	"ttnmapper-gateway-update/types"
)

func FetchStatuses(page int) ([]Openapi.Gateway, error) {
	var gateways []Openapi.Gateway

	client, err := Openapi.NewClientWithResponses("https://mapper.packetbroker.net/api/v2")
	if err != nil {
		return gateways, err
	}

	limit := 1000
	offset := page * limit
	online := true // online only to make responses smaller
	params := Openapi.ListGatewaysParams{
		DistanceWithin: nil,
		Offset:         &offset,
		Limit:          &limit,
		UpdatedSince:   nil,
		Online:         &online,
	}
	//params.DistanceWithin = (*struct {
	//	Openapi.Point `yaml:",inline"`
	//	Distance      float64 `json:"distance"`
	//	Latitude      float32 `json:"latitude"`
	//	Longitude     float32 `json:"longitude"`
	//})(&struct {
	//	Openapi.Point
	//	Distance  float64
	//	Latitude  float32
	//	Longitude float32
	//}{Point: Openapi.Point{Latitude: 22.7, Longitude: 114.234}, Distance: 7500, Latitude: 22.7, Longitude: 114.234})

	listGatewaysResponse, err := client.ListGatewaysWithResponse(context.Background(), &params)
	if err != nil {
		return gateways, err
	}
	if listGatewaysResponse.JSON200 != nil {
		gateways = append(gateways, *listGatewaysResponse.JSON200...)
		if len(*listGatewaysResponse.JSON200) == 0 {
			log.Printf("%s", listGatewaysResponse.Body)
			return gateways, errors.New("response empty")
		} else {
			return gateways, nil
		}
	} else {
		log.Printf("%s", listGatewaysResponse.Body)
		return gateways, errors.New("response nil")
	}
}

func PbGatewayToTtnMapperGateway(gatewayIn Openapi.Gateway) (types.TtnMapperGateway, error) {
	var gatewayOut types.TtnMapperGateway

	if gatewayIn.TenantID == nil {
		return gatewayOut, errors.New("tenant id not set")
	}
	gatewayOut.NetworkId = types.NS_TTS_V3 + "://" + *gatewayIn.TenantID + "@" + gatewayIn.NetID

	// Exception for TTNv2: rewrite NetworkId to one used for Noc and Web sources. Live data uses NS_TTN_V2://ip-addr
	if *gatewayIn.TenantID == "ttnv2" {
		gatewayOut.NetworkId = "thethingsnetwork.org"
	}

	gatewayOut.GatewayId = gatewayIn.Id

	if gatewayIn.Eui != nil {
		gatewayOut.GatewayEui = *gatewayIn.Eui
	} else {
		// If EUI is not set, try and guess from known ID patterns
		// eui-c0ee40ffff29618d
		if len(gatewayIn.Id) == 20 && strings.HasPrefix(gatewayIn.Id, "eui-") {
			gatewayOut.GatewayEui = strings.ToUpper(strings.TrimPrefix(gatewayIn.Id, "eui-"))
		}
		// 00800000a000222e
		_, err := strconv.ParseUint(gatewayIn.Id, 16, 64)
		if err == nil {
			// Is a valid hex number
			if len(gatewayIn.Id) == 16 {
				gatewayOut.GatewayEui = strings.ToUpper(gatewayIn.Id)
			}
		}
	}

	// If gateway is online according to PB, then the last heard is now, else last heard is zero-time-value
	if gatewayIn.Online != nil && *gatewayIn.Online {
		gatewayOut.Time = time.Now().UnixNano()
	}

	if gatewayIn.Location != nil {
		gatewayOut.Latitude = gatewayIn.Location.Latitude
		gatewayOut.Longitude = gatewayIn.Location.Longitude
		if gatewayIn.Location.Altitude != nil {
			altitude := *gatewayIn.Location.Altitude
			gatewayOut.Altitude = int32(altitude)
		}
	}

	// TODO: hdop is not a valid accuracy in metres. Keep an eye on https://github.com/packetbroker/api/issues/32
	//if hdop, ok := gatewayIn.Location.GetHdopOk(); ok {
	//	gatewayOut.LocationAccuracy = int32(*hdop)
	//}

	return gatewayOut, nil
}
