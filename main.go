package main

import (
	"flag"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/streadway/amqp"
	"github.com/tkanos/gonfig"
	"log"
	"net/http"
	_ "net/http/pprof"
	"sync"
	"ttnmapper-gateway-update/types"
	"ttnmapper-gateway-update/utils"
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

	FetchAmqp bool `env:"FETCH_AMQP"` // Should we subscribe to the amqp queue to process live data
	//FetchNoc          bool `env:"FETCH_NOC"`  // Should we periodically fetch gateway statuses from the NOC (TTNv2)

	FetchWeb         bool `env:"FETCH_WEB"`          // Should we periodically fetch gateway statuses from the TTN website (TTNv2 and v3)
	FetchWebInterval int  `env:"FETCH_WEB_INTERVAL"` // How often in seconds should we fetch gateway statuses from the TTN Website

	FetchPacketBroker         bool `env:"FETCH_PACKET_BROKER"`
	FetchPacketBrokerInterval int  `env:"FETCH_PACKET_BROKER_INTERVAL"`

	FetchHelium         bool `env:"FETCH_HELIUM"`
	FetchHeliumInterval int  `env:"FETCH_HELIUM_INTERVAL"`
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

	FetchAmqp: false,
	//FetchNoc:          false,
	FetchWeb:                  false,
	FetchWebInterval:          3600,
	FetchPacketBroker:         false,
	FetchPacketBrokerInterval: 3600,
	FetchHelium:               false,
	FetchHeliumInterval:       86400,
}

var (
	// Prometheus stats
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

	// Other global vars
	gatewayDbIdCache  sync.Map
	rawPacketsChannel = make(chan amqp.Delivery)
	db                *gorm.DB
)

func main() {
	reprocess := flag.Bool("reprocess", false, "Reprocess by fetching gateway statuses from specific endpoints")
	flag.Parse()
	reprocessApis := flag.Args()

	err := gonfig.GetConf("conf.json", &myConfiguration)
	if err != nil {
		log.Println(err)
	}

	log.Printf("[Configuration]\n%s\n", utils.PrettyPrint(myConfiguration)) // output: [UserA, UserB]

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
	// pq: unsupported sslmode "prefer"; only "require" (default), "verify-full", "verify-ca", and "disable" supported - so we disable it
	db, dbErr = gorm.Open("postgres", "host="+myConfiguration.PostgresHost+" port="+myConfiguration.PostgresPort+" user="+myConfiguration.PostgresUser+" dbname="+myConfiguration.PostgresDatabase+" password="+myConfiguration.PostgresPassword+" sslmode=disable")
	if dbErr != nil {
		log.Println("Error connecting to Postgres")
		panic(dbErr.Error())
	}
	defer db.Close()

	if myConfiguration.PostgresDebugLog {
		db.LogMode(true)
	}

	//// Create tables if they do not exist
	//log.Println("Performing auto migrate")
	db.AutoMigrate(
		&types.Gateway{},
		&types.GatewayLocation{},
		&types.GatewayLocationForce{},
	)

	if *reprocess {
		for _, service := range reprocessApis {
			if service == "web" {
				log.Println("Fetching web gateway statuses")
				fetchPacketBrokerStatuses()
			}
			if service == "packetbroker" {
				log.Println("Fetching Packet Broker gateway statuses")
				fetchPacketBrokerStatuses()
			}
			if service == "helium" {
				log.Println("Fetching Helium hotspot statuses")
				fetchHeliumStatuses()
			}
		}
	} else {
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
}
