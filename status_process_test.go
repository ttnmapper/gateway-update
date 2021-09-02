package main

import (
	"encoding/json"
	"github.com/jinzhu/gorm"
	"github.com/tkanos/gonfig"
	"log"
	"testing"
	"time"
	"ttnmapper-gateway-update/types"
	"ttnmapper-gateway-update/utils"
)

var data = "{\"network_type\":\"NS_TTS_V3\",\"network_address\":\"ttn@000013\",\"network_id\":\"NS_TTS_V3://ttn@000013\",\"app_id\":\"mindbird-gps-tracker\",\"dev_id\":\"ttn-mapper-leonardo\",\"dev_eui\":\"0004A30B001FEA80\",\"time\":1622876477587001491,\"port\":1,\"counter\":6,\"frequency\":867900000,\"modulation\":\"LORA\",\"bandwidth\":125000,\"spreading_factor\":7,\"coding_rate\":\"4/5\",\"gateways\":[{\"network_id\":\"NS_TTS_V3://ttn@000013\",\"gtw_id\":\"mindbird-kirchbauna\",\"gtw_eui\":\"DCA632FFFE6AA3DB\",\"antenna_index\":0,\"time\":1622876476346272000,\"timestamp\":2272166741,\"channel\":7,\"rssi\":-95,\"snr\":13.8,\"latitude\":51.242965026571284,\"longitude\":9.428522586822512,\"altitude\":180,\"location_source\":\"SOURCE_REGISTRY\"}],\"latitude\":51.243896,\"longitude\":9.425287,\"altitude\":193.1,\"hdop\":0.91,\"accuracy_source\":\"hdop\",\"userid\":\"redacted\",\"useragent\":\"ttn-lw-application-server/3.13.0\"}"

func SetupDb() {

	err := gonfig.GetConf("conf.json", &myConfiguration)
	if err != nil {
		log.Println(err)
	}

	log.Printf("[Configuration]\n%s\n", utils.PrettyPrint(myConfiguration)) // output: [UserA, UserB]

	// Table name prefixes
	gorm.DefaultTableNameHandler = func(db *gorm.DB, defaultTableName string) string {
		//return "ttnmapper_" + defaultTableName
		return defaultTableName
	}

	var dbErr error
	// pq: unsupported sslmode "prefer"; only "require" (default), "verify-full", "verify-ca", and "disable" supported - so we disable it
	db, dbErr = gorm.Open("postgres", "host="+myConfiguration.PostgresHost+" port="+myConfiguration.PostgresPort+" user="+myConfiguration.PostgresUser+" dbname="+myConfiguration.PostgresDatabase+" password="+myConfiguration.PostgresPassword+" sslmode=disable")
	if dbErr != nil {
		log.Println("Error connecting to Postgres")
		panic(dbErr.Error())
	}

	if myConfiguration.PostgresDebugLog {
		db.LogMode(true)
	}
}

func CloseDb() {
	db.Close()
}

func TestAbs(t *testing.T) {

	SetupDb()

	// The message from amqp is a json string. Unmarshal to ttnmapper uplink struct
	var message types.TtnMapperUplinkMessage
	if err := json.Unmarshal([]byte(data), &message); err != nil {
		t.Errorf(err.Error())
	}

	// Iterate gateways in packet
	for _, gateway := range message.Gateways {
		updateTime := time.Unix(0, message.Time)
		log.Print("AMQP ", "", "\t", gateway.GatewayId+"\t", updateTime)

		// We use the "last heard" on the network
		gateway.Time = message.Time

		// Ignore locations obtained from live data in TTNv2, as it takes 6 hours to update, or is often not set.
		// TODO: we can have an oscillating behaviour between location from metadata and location from other sources. Is this only a V2 issue?
		if message.NetworkType == types.NS_TTN_V2 {
			gateway.Latitude = 0
			gateway.Longitude = 0
			gateway.Altitude = 0
		}

		UpdateGateway(gateway)
	}

	CloseDb()
}

func TestCoordinatesValid(t *testing.T) {
	gateway := types.TtnMapperGateway{Latitude: 22.700000762939453, Longitude: 114.23999786376953}
	valid, reason := CoordinatesValid(gateway)
	if valid {
		t.Fatalf("Shenzen factory should not be valid")
	} else {
		log.Println(reason)
	}
}
