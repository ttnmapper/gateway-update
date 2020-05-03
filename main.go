package main

import (
	"encoding/json"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/streadway/amqp"
	"github.com/tkanos/gonfig"
	"github.com/umahmood/haversine"
	"log"
	"math"
	"net/http"
	"sync"
	"time"
	"ttnmapper-gateway-update/types"
)

type Configuration struct {
	AmqpHost                    string `env:"AMQP_HOST"`
	AmqpPort                    string `env:"AMQP_PORT"`
	AmqpUser                    string `env:"AMQP_USER"`
	AmqpPassword                string `env:"AMQP_PASSWORD"`
	AmqpExchangeRawPackets      string `env:"AMQP_EXHANGE_RAW"`
	AmqpExchangeGatewayStatuses string `env:"AMQP_EXHANGE_STATUS"`
	AmqpQueueRawPackets         string `env:"AMQP_QUEUE"`
	AmqpQueueGatewayStatuses    string `env:"AMQP_QUEUE"`

	PostgresHost          string `env:"POSTGRES_HOST"`
	PostgresPort          string `env:"POSTGRES_PORT"`
	PostgresUser          string `env:"POSTGRES_USER"`
	PostgresPassword      string `env:"POSTGRES_PASSWORD"`
	PostgresDatabase      string `env:"POSTGRES_DATABASE"`
	PostgresDebugLog      bool   `env:"POSTGRES_DEBUG_LOG"`
	PostgresInsertThreads int    `env:"POSTGRES_INSERT_THREADS"`

	PrometheusPort string `env:"PROMETHEUS_PORT"`
}

var myConfiguration = Configuration{
	AmqpHost:                    "localhost",
	AmqpPort:                    "5672",
	AmqpUser:                    "user",
	AmqpPassword:                "password",
	AmqpExchangeRawPackets:      "new_packets",
	AmqpExchangeGatewayStatuses: "gateway_status",
	AmqpQueueRawPackets:         "gateway_updates_raw",
	AmqpQueueGatewayStatuses:    "gateway_updates_status",

	PostgresHost:          "localhost",
	PostgresPort:          "5432",
	PostgresUser:          "username",
	PostgresPassword:      "password",
	PostgresDatabase:      "database",
	PostgresDebugLog:      false,
	PostgresInsertThreads: 1,

	PrometheusPort: "9100",
}

var (
	processedGateways = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ttnmapper_gateway_processed_count",
		Help: "The total number of gateway updates processed",
	})
	newGateways = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ttnmapper_gateway_new_count",
		Help: "The total number of new gateways seen",
	})
	movedGateways = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ttnmapper_gateway_moved_count",
		Help: "The total number of gateways that moved",
	})

	insertDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ttnmapper_gateway_processed_duration",
		Help:    "How long the processing of a gateway status took",
		Buckets: []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1, 1.5, 2, 5, 10, 100, 1000, 10000},
	})
)

var (
	gatewayDbCache sync.Map
)

var rawPacketsChannel = make(chan amqp.Delivery)
var gatewayStatusesChannel = make(chan amqp.Delivery)

func main() {

	err := gonfig.GetConf("conf.json", &myConfiguration)
	if err != nil {
		log.Println(err)
	}

	log.Printf("[Configuration]\n%s\n", prettyPrint(myConfiguration)) // output: [UserA, UserB]

	http.Handle("/metrics", promhttp.Handler())
	go func() {
		err := http.ListenAndServe("0.0.0.0:"+myConfiguration.PrometheusPort, nil)
		if err != nil {
			log.Print(err.Error())
		}
	}()

	// Table name prefixes
	gorm.DefaultTableNameHandler = func(db *gorm.DB, defaultTableName string) string {
		//return "ttnmapper_" + defaultTableName
		return defaultTableName
	}

	db, err := gorm.Open("postgres", "host="+myConfiguration.PostgresHost+" port="+myConfiguration.PostgresPort+" user="+myConfiguration.PostgresUser+" dbname="+myConfiguration.PostgresDatabase+" password="+myConfiguration.PostgresPassword+"")
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	if myConfiguration.PostgresDebugLog {
		db.LogMode(true)
	}

	// Create tables if they do not exist
	log.Println("Performing auto migrate")
	db.AutoMigrate(
		&types.Gateway{},
		&types.GatewayBlacklist{},
		&types.GatewayLocation{},
		&types.GatewayLocationForce{},
	)

	// Start threads to handle Postgres inserts
	log.Println("Starting database insert threads")
	for i := 0; i < myConfiguration.PostgresInsertThreads; i++ {
		go processRawPackets(i+1, db)
	}

	// Start amqp listener on this thread - blocking function
	log.Println("Starting AMQP thread")
	subscribeToRabbitRaw()
}

func subscribeToRabbitRaw() {
	conn, err := amqp.Dial("amqp://" + myConfiguration.AmqpUser + ":" + myConfiguration.AmqpPassword + "@" + myConfiguration.AmqpHost + ":" + myConfiguration.AmqpPort + "/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	err = ch.ExchangeDeclare(
		myConfiguration.AmqpExchangeRawPackets, // name
		"fanout",                               // type
		true,                                   // durable
		false,                                  // auto-deleted
		false,                                  // internal
		false,                                  // no-wait
		nil,                                    // arguments
	)
	failOnError(err, "Failed to declare an exchange")

	q, err := ch.QueueDeclare(
		myConfiguration.AmqpQueueRawPackets, // name
		false,                               // durable
		true,                                // delete when unused
		false,                               // exclusive
		false,                               // no-wait
		nil,                                 // arguments
	)
	failOnError(err, "Failed to declare a queue")

	err = ch.Qos(
		10,    // prefetch count
		0,     // prefetch size
		false, // global
	)
	failOnError(err, "Failed to set queue QoS")

	err = ch.QueueBind(
		q.Name,                                 // queue name
		"",                                     // routing key
		myConfiguration.AmqpExchangeRawPackets, // exchange
		false,
		nil)
	failOnError(err, "Failed to bind a queue")

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")

	// Start thread that listens for new amqp messages
	go func() {
		for d := range msgs {
			log.Print("[a] Packet received")
			rawPacketsChannel <- d
		}
	}()

	log.Printf("Init Complete")
	forever := make(chan bool)
	<-forever
}

func processRawPackets(thread int, db *gorm.DB) {
	// Wait for a message and insert it into Postgres
	for d := range rawPacketsChannel {
		log.Printf(" [%d][p] Processing raw packet", thread)

		// The message form amqp is a json string. Unmarshal to ttnmapper uplink struct
		var message types.TtnMapperUplinkMessage
		if err := json.Unmarshal(d.Body, &message); err != nil {
			log.Print("[%d][p] "+err.Error(), thread)
			continue
		}

		// Last heard time
		seconds := message.Time / 1000000000
		nanos := message.Time % 1000000000
		lastHeard := time.Unix(seconds, nanos)

		// Iterate gateways. We store it flat in the database
		for _, gateway := range message.Gateways {
			log.Printf("  [%d][p] Processing gateway %s", thread, gateway.GatewayId)
			updateGateway(db, gateway, lastHeard)
		}
	}
}

func updateGateway(db *gorm.DB, gateway types.TtnMapperGateway, lastHeard time.Time) {
	gatewayStart := time.Now()

	// Count number of gateways we processed
	processedGateways.Inc()

	lastLocationFound := false
	gatewayMoved := false

	// Find the database ID for this gateway
	gatewayID, err := getGatewayDbID(db, gateway)
	if err != nil {
		failOnError(err, "Can't find gateway ID")
	}

	// Update last seen in Gateways table
	pgGateway := types.Gateway{ID: gatewayID, LastHeard: lastHeard}
	db.Model(&pgGateway).Update(pgGateway)

	// Check if this gateway is blacklisted
	if isGatewayBlacklisted(db, gateway) {
		log.Println("    Gateway is blacklisted")
		return
	}

	// Check if the coordinates should be isForced to a specific location
	if isForced, forcedCoordinates := isCoordinatesForced(db, gatewayID); isForced == true {
		log.Println("    Gateway coordinates forced")
		gateway.Latitude = forcedCoordinates.Latitude
		gateway.Longitude = forcedCoordinates.Longitude
	}

	// Check if the provided new coordinates are valid
	if !coordinatesValid(gateway) {
		log.Println("    Gateway coordinates invalid")
		return
	}

	// Find the last known location for this gateway
	gatewayLastLocation := getGatewayLastLocation(db, gatewayID)
	if err != nil {
		failOnError(err, "Can't find last gateway location")
	}

	// Check if we found a last location or not
	if gatewayLastLocation.ID != 0 {
		lastLocationFound = true
		oldLocation := haversine.Coord{Lat: gatewayLastLocation.Latitude, Lon: gatewayLastLocation.Longitude} // Oxford, UK
		newLocation := haversine.Coord{Lat: gateway.Latitude, Lon: gateway.Longitude}                         // Turin, Italy
		_, km := haversine.Distance(oldLocation, newLocation)

		// Did it move more than 100m
		if km > 0.1 {
			gatewayMoved = true
		}

		// TODO: because we can add historical data, how are we going to make sure the "installed at" of the currently used location is backdated?
	}

	// A new gateway, insert new entry, and maybe publish on Twitter/Slack?
	if lastLocationFound == false {
		log.Println("    New gateway")
		newGateways.Inc()

		// Insert a new location entry for the history of this gateway
		insertNewLocationForGateway(db, gatewayID, lastHeard, gateway)
		//updateGatewayLocationAndSource(db, gatewayID, lastHeard, gateway)
	}

	// Gateway moved, insert new entry
	if gatewayMoved {
		log.Println("    Gateway moved")
		movedGateways.Inc()
		insertNewLocationForGateway(db, gatewayID, lastHeard, gateway)
		//updateGatewayLocationAndSource(db, gatewayID, lastHeard, gateway)
	}

	// TODO: temporarily always update the coordinates in the gateways table
	updateGatewayLocationAndSource(db, gatewayID, lastHeard, gateway)

	// Prometheus stats
	gatewayElapsed := time.Since(gatewayStart)
	insertDuration.Observe(float64(gatewayElapsed.Nanoseconds()) / 1000.0 / 1000.0) //nanoseconds to milliseconds
}

func getGatewayDbID(db *gorm.DB, gateway types.TtnMapperGateway) (uint, error) {
	// GatewayID
	i, ok := gatewayDbCache.Load(gateway.GatewayId)
	if ok {
		return i.(uint), nil
	} else {
		gatewayDb := types.Gateway{GtwId: gateway.GatewayId, GtwEui: gateway.GatewayEui}
		err := db.FirstOrCreate(&gatewayDb, &gatewayDb).Error
		if err != nil {
			return 0, err
		}
		gatewayDbCache.Store(gateway.GatewayId, gatewayDb.ID)
		return gatewayDb.ID, nil
	}
}

func isGatewayBlacklisted(db *gorm.DB, gateway types.TtnMapperGateway) bool {

	// First check using gateway ID
	blacklistGateway := types.GatewayBlacklist{GtwId: gateway.GatewayId}
	db.First(&blacklistGateway, &blacklistGateway)
	if blacklistGateway.ID != 0 {
		return true
	}

	// Then check using gatewayEUI
	//blacklistGateway = types.GatewayBlacklist{GtwEui: gateway.GatewayEui}
	//db.First(&blacklistGateway, &blacklistGateway)
	//if blacklistGateway.ID != 0 {
	//	return true
	//}

	return false
}

func isCoordinatesForced(db *gorm.DB, gatewayID uint) (bool, types.GatewayLocationForce) {
	forcedCoords := types.GatewayLocationForce{GatewayID: gatewayID}
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

func getGatewayLastLocation(db *gorm.DB, gatewayID uint) types.GatewayLocation {
	// returns latitude, longitude, error

	gatewayLocation := types.GatewayLocation{GatewayID: gatewayID}
	db.Order("installed_at DESC").First(&gatewayLocation, &gatewayLocation)

	return gatewayLocation
}

func insertNewLocationForGateway(db *gorm.DB, gatewayID uint, installedAt time.Time, gateway types.TtnMapperGateway) {
	newLocation := types.GatewayLocation{
		GatewayID:   gatewayID,
		InstalledAt: installedAt,
		Latitude:    gateway.Latitude,
		Longitude:   gateway.Longitude,
	}
	db.Create(&newLocation)
}

func updateGatewayLocationAndSource(db *gorm.DB, gatewayID uint, lastHeard time.Time, gateway types.TtnMapperGateway) {
	pgGateway := types.Gateway{
		ID:        gatewayID,
		Latitude:  &gateway.Latitude,
		Longitude: &gateway.Longitude,
		Altitude:  &gateway.Altitude,
		LastHeard: lastHeard,
	}

	if gateway.LocationAccuracy != 0 {
		pgGateway.LocationAccuracy = &gateway.LocationAccuracy
	}
	if gateway.LocationSource != "" {
		pgGateway.LocationSource = &gateway.LocationSource
	}

	db.Model(&pgGateway).Update(pgGateway)
}
