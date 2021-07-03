package main

import (
	"github.com/streadway/amqp"
	"log"
	"time"
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
	nocTicker := time.NewTicker(time.Duration(myConfiguration.StatusFetchInterval) * time.Second)
	webTicker := time.NewTicker(time.Duration(myConfiguration.StatusFetchInterval) * time.Second)
	pbTicker := time.NewTicker(time.Duration(myConfiguration.StatusFetchInterval) * time.Second)

	go func() {
		for {
			select {
			case <-nocTicker.C:
				if myConfiguration.FetchNoc {
					go fetchNocStatuses()
				}
			case <-webTicker.C:
				if myConfiguration.FetchWeb {
					go fetchWebStatuses()
				}
			case <-pbTicker.C:
				if myConfiguration.FetchPacketBroker {
					go fetchPacketBrokerStatuses()
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

	gateways, err := web.FetchWebStatuses()
	if err != nil {
		log.Println(err.Error())
	} else {
		for _, gateway := range gateways {
			ttnMapperGateway := web.WebGatewayToTtnMapperGateway(*gateway)
			UpdateGateway(ttnMapperGateway)
		}
	}

	busyFetchingWeb = false
}

var busyFetchingPacketBroker = false

func fetchPacketBrokerStatuses() {
	if busyFetchingPacketBroker {
		return
	}
	busyFetchingPacketBroker = true

	gateways, err := packet_broker.FetchStatuses()
	if err != nil {
		log.Println(err.Error())
	} else {
		for _, gateway := range gateways {
			ttnMapperGateway, err := packet_broker.PbGatewayToTtnMapperGateway(gateway)
			if err == nil {
				UpdateGateway(ttnMapperGateway)
			}
		}
	}

	busyFetchingPacketBroker = false
}
