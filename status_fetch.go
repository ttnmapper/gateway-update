package main

import (
	"encoding/json"
	"github.com/streadway/amqp"
	"log"
	"net/http"
	"time"
	"ttnmapper-gateway-update/types"
	"ttnmapper-gateway-update/utils"
)

func subscribeToRabbitRaw() {
	// Start thread that listens for new amqp messages
	go func() {
		conn, err := amqp.Dial("amqp://" + myConfiguration.AmqpUser + ":" + myConfiguration.AmqpPassword + "@" + myConfiguration.AmqpHost + ":" + myConfiguration.AmqpPort + "/")
		utils.FailOnError(err, "Failed to connect to RabbitMQ")
		defer conn.Close()

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

		for d := range msgs {
			rawPacketsChannel <- d
		}
	}()

	// Start the thread that processes new amqp messages
	go func() {
		processRawPackets()
	}()
}

func startPeriodicFetchers() {
	nocTicker := time.NewTicker(time.Duration(myConfiguration.StatusFetchInterval) * time.Second)
	webTicker := time.NewTicker(time.Duration(myConfiguration.StatusFetchInterval) * time.Second)

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
	log.Println("Fetching NOC statuses")

	httpClient := http.Client{
		Timeout: time.Second * 60, // Maximum of 1 minute
	}

	req, err := http.NewRequest(http.MethodGet, myConfiguration.NocUrl, nil)
	if err != nil {
		log.Println(err)
		return
	}

	req.Header.Set("User-Agent", "ttnmapper-update-gateway")

	res, err := httpClient.Do(req)
	if err != nil {
		log.Println(err)
		return
	}

	defer res.Body.Close()

	nocData := types.NocStatus{}
	err = json.NewDecoder(res.Body).Decode(&nocData)
	if err != nil {
		log.Println(err)
		return
	}

	count := 0
	total := len(nocData.Statuses)
	for gatewayId, gateway := range nocData.Statuses {
		count++
		log.Print("NOC ", count, "/", total, "\t", gatewayId+"\t", gateway.Timestamp)
		processNocGateway(gatewayId, gateway)
	}

	log.Println("Fetching NOC statuses done")
	busyFetchingNoc = false
}

var busyFetchingWeb = false

func fetchWebStatuses() {
	if busyFetchingWeb {
		return
	}
	busyFetchingWeb = true
	log.Println("Fetching web statuses")

	httpClient := http.Client{
		Timeout: time.Second * 60, // Maximum of 1 minute
	}

	req, err := http.NewRequest(http.MethodGet, myConfiguration.WebUrl, nil)
	if err != nil {
		log.Println(err)
		return
	}

	req.Header.Set("User-Agent", "ttnmapper-update-gateway")

	res, err := httpClient.Do(req)
	if err != nil {
		log.Println(err)
		return
	}

	defer res.Body.Close()

	webData := map[string]*types.WebGateway{}
	err = json.NewDecoder(res.Body).Decode(&webData)
	if err != nil {
		log.Println(err)
		return
	}

	count := 0
	total := len(webData)
	for gatewayId, gateway := range webData {
		count++
		log.Print("WEB ", count, "/", total, "\t", gatewayId+"\t", gateway.LastSeen)
		processWebGateway(*gateway)
	}

	log.Println("Fetching web statuses done")
	busyFetchingWeb = false
}
