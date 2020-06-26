package main

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/streadway/amqp"
	"github.com/tkanos/gonfig"
	"log"
	"net/http"
	"sync"
	"ttnmapper-gateway-update/types"
)

type Configuration struct {
	AmqpHost                 string `env:"AMQP_HOST"`
	AmqpPort                 string `env:"AMQP_PORT"`
	AmqpUser                 string `env:"AMQP_USER"`
	AmqpPassword             string `env:"AMQP_PASSWORD"`
	AmqpExchangeRawPackets   string `env:"AMQP_EXHANGE_RAW"`
	AmqpQueueRawPackets      string `env:"AMQP_QUEUE"`
	AmqpExchangeGatewayMoved string `env:"AMQP_EXCHANGE_GATEWAY_MOVED"`

	PostgresHost     string `env:"POSTGRES_HOST"`
	PostgresPort     string `env:"POSTGRES_PORT"`
	PostgresUser     string `env:"POSTGRES_USER"`
	PostgresPassword string `env:"POSTGRES_PASSWORD"`
	PostgresDatabase string `env:"POSTGRES_DATABASE"`
	PostgresDebugLog bool   `env:"POSTGRES_DEBUG_LOG"`

	PrometheusPort string `env:"PROMETHEUS_PORT"`

	FetchAmqp    bool   `env:"FETCH_AMQP"` // Should we subscribe to the amqp queue to process live data
	FetchNoc     bool   `env:"FETCH_NOC"`  // Should we periodically fetch gateway statuses from the NOC (TTNv2)
	NocUrl       string `env:"NOC_URL"`
	NocBasicAuth bool   `env:"NOC_BASIC_AUTH"`
	NocUsername  string `env:"NOC_USERNAME"`
	NocPassword  string `env:"NOC_PASSWORD"`
	FetchWeb     bool   `env:"FETCH_WEB"` // Should we periodivally fetch gateway statuses from the TTN website (TTNv2 and v3)
	WebUrl       string `env:"WEB_URL"`
	// TODO: Fetch gateway statuses from V3 API or Packet Broker API

	StatusFetchInterval int `env:"FETCH_INTERVAL"` // How often in seconds should we fetch gateway statuses from the NOC and the TTN Website
}

var myConfiguration = Configuration{
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

var (
	processedGateways = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ttnmapper_gateway_processed_count",
		Help: "The total number of gateway updates processed",
	})
	updatedGateways = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ttnmapper_gateway_updated_count",
		Help: "The total number of gateways updated",
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
	gatewayDbIdCache  sync.Map
	rawPacketsChannel = make(chan amqp.Delivery)
	db                *gorm.DB
)

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

	var dbErr error
	db, dbErr = gorm.Open("postgres", "host="+myConfiguration.PostgresHost+" port="+myConfiguration.PostgresPort+" user="+myConfiguration.PostgresUser+" dbname="+myConfiguration.PostgresDatabase+" password="+myConfiguration.PostgresPassword+" sslmode=prefer")
	if dbErr != nil {
		log.Println("Error connecting to Postgres")
		panic(dbErr.Error())
	}
	defer db.Close()

	if myConfiguration.PostgresDebugLog {
		db.LogMode(true)
	}

	// Create tables if they do not exist
	log.Println("Performing auto migrate")
	db.AutoMigrate(
		&types.Gateway{},
		&types.GatewayLocation{},
		&types.GatewayLocationForce{},
	)

	// Start amqp listener on this thread - blocking function
	if myConfiguration.FetchAmqp {
		log.Println("Starting AMQP thread")
		subscribeToRabbitRaw()
	}

	// Periodic status fetchers
	startPeriodicFetchers()

	log.Printf("Init Complete")
	forever := make(chan bool)
	<-forever
}
