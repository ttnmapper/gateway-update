package main

import (
	"github.com/jinzhu/gorm"
	"github.com/tkanos/gonfig"
	"log"
	"testing"
	"ttnmapper-gateway-update/utils"
)

func initTests() {
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

//func TestFectNocStatuses(t *testing.T) {
//	initTests()
//	fetchNocStatuses()
//}

//func TestFetchWebStatuses(t *testing.T) {
//	initTests()
//	fetchWebStatuses()
//}

func TestFetchPacketBrokerStatuses(t *testing.T) {
	initTests()
	fetchPacketBrokerStatuses()
}

func TestFetchHeliumStatuses(t *testing.T) {
	initTests()
	fetchHeliumStatuses()
}
