package main

import (
	"encoding/json"
	"github.com/umahmood/haversine"
	"log"
	"math"
	"time"
	"ttnmapper-gateway-update/types"
)

func processRawPackets(thread int) {
	// Wait for a message and insert it into Postgres
	for d := range rawPacketsChannel {
		log.Printf(" [%d][p] Processing raw packet", thread)

		// The message form amqp is a json string. Unmarshal to ttnmapper uplink struct
		var message types.TtnMapperUplinkMessage
		if err := json.Unmarshal(d.Body, &message); err != nil {
			log.Print("[%d][p] "+err.Error(), thread)
			continue
		}

		// Iterate gateways. We store it flat in the database
		for _, gateway := range message.Gateways {
			log.Printf("  [%d][p] Processing gateway %s", thread, gateway.GatewayId)
			gateway.Time = message.Time

			// Packet broker metadata will provide network id. For now assume TTN
			gateway.NetorkId = "thethingsnetwork.org"

			updateGateway(gateway)
		}
	}
}

func processNocGateway(gatewayId string, gateway types.NocGateway) {
	ttnMapperGateway := types.TtnMapperGateway{}
	ttnMapperGateway.GatewayId = gatewayId
	ttnMapperGateway.Time = gateway.Timestamp.Unix()
	ttnMapperGateway.Latitude = gateway.Location.Latitude
	ttnMapperGateway.Longitude = gateway.Location.Longitude
	//ttnMapperGateway.Altitude = int32(gateway.Location.Altitude)
	//ttnMapperGateway.Description = gateway.Description

	// Assume NOC lists only TTN gateways. Need to check this as a private V2 network can also have a NOC
	ttnMapperGateway.NetorkId = "thethingsnetwork.org"

	updateGateway(ttnMapperGateway)
}

func processWebGateway(gateway types.WebGateway) {
	ttnMapperGateway := types.TtnMapperGateway{}
	ttnMapperGateway.GatewayId = gateway.ID
	ttnMapperGateway.Time = gateway.LastSeen.Unix()
	ttnMapperGateway.Latitude = gateway.Location.Latitude
	ttnMapperGateway.Longitude = gateway.Location.Longitude
	ttnMapperGateway.Altitude = int32(gateway.Location.Altitude)
	ttnMapperGateway.Description = gateway.Description

	// Website lists only TTN gateways
	ttnMapperGateway.NetorkId = "thethingsnetwork.org"

	updateGateway(ttnMapperGateway)
}

func updateGateway(gateway types.TtnMapperGateway) {
	gatewayStart := time.Now()

	// Count number of gateways we processed
	processedGateways.Inc()

	gatewayMoved := false

	// Last heard time
	seconds := gateway.Time / 1000000000
	nanos := gateway.Time % 1000000000
	lastHeard := time.Unix(seconds, nanos)

	// Find the database IDs for this gateway and it's antennas
	gatewayDbId, err := getGatewayDbId(gateway)
	if err != nil {
		failOnError(err, "Can't find gateway ID")
	}

	// Update last seen in Gateways table
	pgGateway := types.Gateway{ID: gatewayDbId, LastHeard: lastHeard}
	db.Model(&pgGateway).Update(pgGateway)

	// Update EUI if it is set
	if gateway.GatewayEui != "" {
		pgGateway := types.Gateway{ID: gatewayDbId, GatewayEui: &gateway.GatewayEui}
		db.Model(&pgGateway).Update(pgGateway)
	}

	// Update description if it is set
	if gateway.Description != "" {
		pgGateway := types.Gateway{ID: gatewayDbId, Description: &gateway.Description}
		db.Model(&pgGateway).Update(pgGateway)
	}

	// Check if the coordinates should be forced to a specific location
	if isForced, forcedCoordinates := isCoordinatesForced(gateway); isForced == true {
		log.Println("    Gateway coordinates forced")
		gateway.Latitude = forcedCoordinates.Latitude
		gateway.Longitude = forcedCoordinates.Longitude
	}

	// Check if the provided coordinates are valid
	if !coordinatesValid(gateway) {
		log.Println("    Gateway coordinates invalid. Forcing to 0,0.")
		gateway.Latitude = 0
		gateway.Longitude = 0
	}

	// Find the last known location for this gateway
	gatewayLastLocation := getGatewayLastLocation(gateway)
	if err != nil {
		failOnError(err, "Can't find last gateway location")
	}

	// Check if we found a last location
	if gatewayLastLocation.ID != 0 {
		oldLocation := haversine.Coord{Lat: gatewayLastLocation.Latitude, Lon: gatewayLastLocation.Longitude}
		newLocation := haversine.Coord{Lat: gateway.Latitude, Lon: gateway.Longitude}
		_, km := haversine.Distance(oldLocation, newLocation)

		// Did it move more than 100m
		if km > 0.1 {
			movedGateways.Inc()
			gatewayMoved = true
		}
	} else {
		// A new gateway, insert new entry, and maybe publish on Twitter/Slack?
		log.Println("    New gateway")
		newGateways.Inc()
		gatewayMoved = true
	}

	// Gateway moved, insert new entry
	if gatewayMoved {
		log.Println("    Gateway moved")
		insertNewLocationForGateway(gateway, lastHeard)
	}

	// TODO: temporarily always update the coordinates in the gateways table
	if gatewayMoved {
		updateGatewayLocation(gatewayDbId, lastHeard, gateway)
	}

	// Prometheus stats
	gatewayElapsed := time.Since(gatewayStart)
	insertDuration.Observe(float64(gatewayElapsed.Nanoseconds()) / 1000.0 / 1000.0) //nanoseconds to milliseconds
}

/*
Takes a TTN Mapper Gateway and search for it in the database and return the database entry id
*/
func getGatewayDbId(gateway types.TtnMapperGateway) (uint, error) {

	gatewayIndexer := types.GatewayIndexer{GatewayId: gateway.GatewayId, NetworkId: gateway.NetorkId}
	i, ok := gatewayDbCache.Load(gatewayIndexer)
	if ok {
		dbId := i.(uint)
		return dbId, nil

	} else {
		gatewayDb := types.Gateway{GatewayId: gateway.GatewayId, NetworkId: gateway.NetorkId}
		err := db.FirstOrCreate(&gatewayDb, &gatewayDb).Error
		if err != nil {
			return 0, err
		}

		gatewayDbCache.Store(gatewayIndexer, gatewayDb.ID)
		return gatewayDb.ID, nil
	}
}

func isCoordinatesForced(gateway types.TtnMapperGateway) (bool, types.GatewayLocationForce) {
	forcedCoords := types.GatewayLocationForce{GatewayId: gateway.GatewayId, NetworkId: gateway.NetorkId}
	db.First(&forcedCoords, &forcedCoords)
	if forcedCoords.ID != 0 {
		return true, forcedCoords
	} else {
		return false, forcedCoords
	}
}

func coordinatesValid(gateway types.TtnMapperGateway) bool {
	if math.Abs(gateway.Latitude) < 1 {
		return false
	}
	if math.Abs(gateway.Longitude) < 1 {
		return false
	}
	if math.Abs(gateway.Latitude) > 90 {
		return false
	}
	if math.Abs(gateway.Longitude) > 180 {
		return false
	}

	// Default SCG location
	if gateway.Latitude == 52.0 && gateway.Longitude == 6.0 {
		log.Println("      Single channel gateway default coordinates")
		return false
	}

	// Default Lorier LR2 location
	if gateway.Latitude == 10.0 && gateway.Longitude == 20.0 {
		log.Println("      Lorier LR2 default coordinates")
		return false
	}

	// Ukrainian hack
	if gateway.Latitude == 50.008724 && gateway.Longitude == 36.215805 {
		log.Println("      Ukrainian hack coordinates")
		return false
	}

	return true
}

func getGatewayLastLocation(gateway types.TtnMapperGateway) types.GatewayLocation {
	gatewayLocation := types.GatewayLocation{GatewayId: gateway.GatewayId, NetworkId: gateway.NetorkId}
	db.Order("installed_at DESC").First(&gatewayLocation, &gatewayLocation)

	return gatewayLocation
}

func insertNewLocationForGateway(gateway types.TtnMapperGateway, installedAt time.Time) {
	newLocation := types.GatewayLocation{
		GatewayId:   gateway.GatewayId,
		NetworkId:   gateway.NetorkId,
		InstalledAt: installedAt,
		Latitude:    gateway.Latitude,
		Longitude:   gateway.Longitude,
	}
	db.Create(&newLocation)
}

func updateGatewayLocation(gatewayDbId uint, lastHeard time.Time, gateway types.TtnMapperGateway) {
	pgGateway := types.Gateway{
		ID:        gatewayDbId,
		Latitude:  &gateway.Latitude,
		Longitude: &gateway.Longitude,
		LastHeard: lastHeard,
	}

	// NOC doesn't provide altitude, so do not update it
	if gateway.Altitude != 0 {
		pgGateway.Altitude = &gateway.Altitude
	}
	if gateway.LocationAccuracy != 0 {
		pgGateway.LocationAccuracy = &gateway.LocationAccuracy
	}
	if gateway.LocationSource != "" {
		pgGateway.LocationSource = &gateway.LocationSource
	}

	db.Model(&pgGateway).Update(pgGateway)
}
