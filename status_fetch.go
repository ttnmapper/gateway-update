package main

import (
	"github.com/streadway/amqp"
	"log"
	"time"
	"ttnmapper-gateway-update/helium"
	"ttnmapper-gateway-update/noc"
	"ttnmapper-gateway-update/packet_broker"
	"ttnmapper-gateway-update/utils"
	"ttnmapper-gateway-update/web"
)

func subscribeToRabbitRaw() {
	// Start thread that listens for new amqp messages
	go func() {
		conn, err := amqp.Dial("amqp://" + myConfiguration.AmqpUser + ":" + myConfiguration.AmqpPassword + "@" + myConfiguration.AmqpHost + ":" + myConfiguration.AmqpPort + "/")
		utils.FailOnError(err, "Failed to connect to RabbitMQ")
		defer conn.Close()

		// Create a channel for errors
		notify := conn.NotifyClose(make(chan *amqp.Error)) //error channel

		ch, err := conn.Channel()
		utils.FailOnError(err, "Failed to open a channel")
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
		utils.FailOnError(err, "Failed to declare an exchange")

		q, err := ch.QueueDeclare(
			myConfiguration.AmqpQueueRawPackets, // name
			false,                               // durable
			false,                               // delete when unused
			false,                               // exclusive
			false,                               // no-wait
			nil,                                 // arguments
		)
		utils.FailOnError(err, "Failed to declare a queue")

		err = ch.Qos(
			10,    // prefetch count
			0,     // prefetch size
			false, // global
		)
		utils.FailOnError(err, "Failed to set queue QoS")

		err = ch.QueueBind(
			q.Name,                                 // queue name
			"",                                     // routing key
			myConfiguration.AmqpExchangeRawPackets, // exchange
			false,
			nil)
		utils.FailOnError(err, "Failed to bind a queue")

		msgs, err := ch.Consume(
			q.Name, // queue
			"",     // consumer
			true,   // auto-ack
			false,  // exclusive
			false,  // no-local
			false,  // no-wait
			nil,    // args
		)
		utils.FailOnError(err, "Failed to register a consumer")

		log.Println("AMQP started")

	waitForMessages:
		for {
			select {
			case err := <-notify:
				if err != nil {
					log.Println(err.Error())
				}
				break waitForMessages
			case d := <-msgs:
				//log.Printf(" [a] Packet received")
				rawPacketsChannel <- d
			}
		}

		log.Fatal("Subscribe channel closed")
	}()

	// Start the thread that processes new amqp messages
	go func() {
		processRawPackets()
	}()
}

func startPeriodicFetchers() {
	//nocTicker := time.NewTicker(time.Duration(myConfiguration.FetchNocInterval) * time.Second)
	webTicker := time.NewTicker(time.Duration(myConfiguration.FetchWebInterval) * time.Second)
	pbTicker := time.NewTicker(time.Duration(myConfiguration.FetchPacketBrokerInterval) * time.Second)
	heliumTicker := time.NewTicker(time.Duration(myConfiguration.FetchHeliumInterval) * time.Second)

	go func() {
		for {
			select {
			//case <-nocTicker.C:
			//	if myConfiguration.FetchNoc {
			//		go fetchNocStatuses()
			//	}
			case <-webTicker.C:
				if myConfiguration.FetchWeb {
					go fetchWebStatuses()
				}
			case <-pbTicker.C:
				if myConfiguration.FetchPacketBroker {
					go fetchPacketBrokerStatuses()
				}
			case <-heliumTicker.C:
				if myConfiguration.FetchHelium {
					go fetchHeliumStatuses()
				}
			}
		}
	}()
}

var busyFetchingNoc = false

func fetchNocStatuses() {
	if busyFetchingNoc {
		return
	}
	busyFetchingNoc = true

	gateways, err := noc.FetchNocStatuses()
	if err != nil {
		log.Println(err.Error())
	} else {
		for id, gateway := range gateways {
			ttnMapperGateway := noc.NocGatewayToTtnMapperGateway(id, gateway)
			log.Print("NOC ", "", "\t", ttnMapperGateway.GatewayId+"\t", ttnMapperGateway.Time)
			UpdateGateway(ttnMapperGateway)
		}
	}

	busyFetchingNoc = false
}

var busyFetchingWeb = false

func fetchWebStatuses() {
	if busyFetchingWeb {
		return
	}
	busyFetchingWeb = true

	gatewayCount := 0
	gateways, err := web.FetchWebStatuses()
	if err != nil {
		log.Println(err.Error())
	} else {
		for _, gateway := range gateways {
			gatewayCount++
			ttnMapperGateway := web.WebGatewayToTtnMapperGateway(*gateway)
			log.Print("WEB ", "", "\t", ttnMapperGateway.GatewayId+"\t", ttnMapperGateway.Time)
			UpdateGateway(ttnMapperGateway)
		}
	}

	log.Printf("Fetched %d gateways from TTN website", gatewayCount)
	busyFetchingWeb = false
}

/*
Fetching on 2021-11-17 took
real	4m5.426s
user	0m3.360s
sys	0m4.620s
*/
var busyFetchingPacketBroker = false

func fetchPacketBrokerStatuses() {
	if busyFetchingPacketBroker {
		return
	}
	busyFetchingPacketBroker = true

	gatewayCount := 0
	page := 0
	for {

		gateways, err := packet_broker.FetchStatuses(page)
		if err != nil {
			log.Println(err.Error())
			break
		} else {
			for _, gateway := range gateways {
				gatewayCount++
				ttnMapperGateway, err := packet_broker.PbGatewayToTtnMapperGateway(gateway)
				if err == nil {
					log.Print("PB ", "", "\t", ttnMapperGateway.GatewayId+"\t", ttnMapperGateway.Time)
					UpdateGateway(ttnMapperGateway)
				}
			}
		}
		page++
	}

	log.Printf("Fetched %d gateways from Packet Broker", gatewayCount)
	busyFetchingPacketBroker = false
}

/*
Fetching Helium on 2021-11-17 took
real    306m29.503s
user    1m47.352s
sys     3m13.356s
*/
var busyFetchingHelium = false

func fetchHeliumStatuses() {
	if busyFetchingHelium {
		return
	}
	busyFetchingHelium = true

	hotspotCount := 0
	cursor := ""
	for {
		response, err := helium.FetchStatuses(cursor)
		if err != nil {
			log.Println(err.Error())
			break
		}

		log.Printf("HELIUM %d hotspots\n", len(response.Data))

		for _, hotspot := range response.Data {
			hotspotCount++
			ttnMapperGateway, err := helium.HeliumHotspotToTtnMapperGateway(hotspot)
			if err == nil {
				//log.Print("HELIUM ", "", "\t", ttnMapperGateway.GatewayId+"\t", ttnMapperGateway.Time)
				UpdateGateway(ttnMapperGateway)
			}
		}

		cursor = response.Cursor
		if cursor == "" {
			log.Println("Cursor empty")
			break
		}
		if len(response.Data) == 0 {
			log.Println("No hotspots in response")
			break
		}
	}

	log.Printf("Fetched %d hotspots from Helium", hotspotCount)
	busyFetchingHelium = false
}
