package tests

import (
	"github.com/jinzhu/gorm"
	"log"
	"testing"
	"time"
	"ttnmapper-gateway-update"
	"ttnmapper-gateway-update/types"
	"ttnmapper-gateway-update/utils"
)

var myConfiguration = main.Configuration{
	AmqpHost:                 "localhost",
	AmqpPort:                 "5672",
	AmqpUser:                 "user",
	AmqpPassword:             "password",
	AmqpExchangeRawPackets:   "new_packets",
	AmqpQueueRawPackets:      "gateway_updates_raw",
	AmqpExchangeGatewayMoved: "gateway_moved",

	PostgresHost:     "localhost",
	PostgresPort:     "5432",
	PostgresUser:     "username",
	PostgresPassword: "password",
	PostgresDatabase: "database",
	PostgresDebugLog: false,

	PrometheusPort: "9100",

	FetchAmqp:    false,
	FetchNoc:     true,
	NocUrl:       "http://noc.thethingsnetwork.org:8085/api/v2/gateways",
	NocBasicAuth: false,
	NocUsername:  "",
	NocPassword:  "",
	FetchWeb:     false,
	WebUrl:       "https://www.thethingsnetwork.org/gateway-data/",

	StatusFetchInterval: 10,
}

func TestUpdateGateway(t *testing.T) {

	log.Printf("[Configuration]\n%s\n", utils.PrettyPrint(myConfiguration)) // output: [UserA, UserB]

	// Init db connection
	gorm.DefaultTableNameHandler = func(db *gorm.DB, defaultTableName string) string {
		return defaultTableName
	}

	var dbErr error
	db, dbErr := gorm.Open("postgres", "host="+myConfiguration.PostgresHost+" port="+myConfiguration.PostgresPort+" user="+myConfiguration.PostgresUser+" dbname="+myConfiguration.PostgresDatabase+" password="+myConfiguration.PostgresPassword+"")
	if dbErr != nil {
		panic(dbErr.Error())
	}
	defer db.Close()

	// Make a gateway object to test with
	ttnMapperGateway := types.TtnMapperGateway{}
	ttnMapperGateway.NetworkId = "thethingsnetwork.org"
	ttnMapperGateway.GatewayId = "stellenbosch-technopark"
	ttnMapperGateway.Time = time.Now().UnixNano()
	ttnMapperGateway.Latitude = 34
	ttnMapperGateway.Longitude = 19
	//ttnMapperGateway.Altitude = int32(gateway.Location.Altitude)
	//ttnMapperGateway.Description = gateway.Description

	for i := 1; i <= 10; i++ {
		ttnMapperGateway.Time = time.Now().UnixNano()
		main.UpdateGateway(ttnMapperGateway)
	}
}
