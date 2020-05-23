package main

import (
	"github.com/jinzhu/gorm"
	"github.com/tkanos/gonfig"
	"log"
	"testing"
	"time"
	"ttnmapper-gateway-update/types"
)

//func TestMain(m *testing.M) {
//	main()
//}

func TestUpdateGateway(t *testing.T) {

	// Read config
	err := gonfig.GetConf("conf.json", &myConfiguration)
	if err != nil {
		log.Println(err)
	}

	log.Printf("[Configuration]\n%s\n", prettyPrint(myConfiguration)) // output: [UserA, UserB]

	// Init db connection
	gorm.DefaultTableNameHandler = func(db *gorm.DB, defaultTableName string) string {
		return defaultTableName
	}

	var dbErr error
	db, dbErr = gorm.Open("postgres", "host="+myConfiguration.PostgresHost+" port="+myConfiguration.PostgresPort+" user="+myConfiguration.PostgresUser+" dbname="+myConfiguration.PostgresDatabase+" password="+myConfiguration.PostgresPassword+"")
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

	updateGateway(ttnMapperGateway)
	updateGateway(ttnMapperGateway)
}
