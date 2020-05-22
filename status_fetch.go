package main

import (
	"encoding/json"
	"github.com/streadway/amqp"
	"log"
	"net/http"
	"time"
	"ttnmapper-gateway-update/types"
)

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
}

func startPeriodicFetchers() {
	nocTicker := time.NewTicker(time.Duration(myConfiguration.StatusFetchInterval) * time.Second)
	webTicker := time.NewTicker(time.Duration(myConfiguration.StatusFetchInterval) * time.Second)

	shouldFetchNoc := false
	shouldFetchWeb := false

	go func() {
		for {
			select {
			case <-nocTicker.C:
				shouldFetchNoc = true
			case <-webTicker.C:
				shouldFetchWeb = true
			}
		}
	}()

	go func() {
		for {
			if shouldFetchNoc && myConfiguration.FetchNoc {
				shouldFetchNoc = false
				fetchNocStatuses()
			}
			if shouldFetchWeb && myConfiguration.FetchWeb {
				shouldFetchWeb = false
				fetchWebStatuses()
			}
		}
	}()
}

func fetchNocStatuses() {
	log.Println("Fetching NOC statuses")

	httpClient := http.Client{
		Timeout: time.Second * 60, // Maximum of 2 secs
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

	for gatewayId, gateway := range nocData.Statuses {
		log.Print(gatewayId+": ", gateway.Timestamp)
		processNocGateway(gatewayId, gateway)
	}

	log.Println("Fetching NOC statuses done")
}

func fetchWebStatuses() {
	log.Println("Fetching web statuses")

	httpClient := http.Client{
		Timeout: time.Second * 60, // Maximum of 2 secs
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

	for gatewayId, gateway := range webData {
		log.Print(gatewayId+": ", gateway.LastSeen)
		processWebGateway(*gateway)
	}

	log.Println("Fetching web statuses done")
}
