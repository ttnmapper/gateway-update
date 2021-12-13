package web

import (
	"log"
	"testing"
)

func TestFetchWebStatuses(t *testing.T) {
	gateways, err := FetchWebStatuses()
	if err != nil {
		t.Fatalf(err.Error())
	}
	for _, gateway := range gateways {
		//log.Println(utils.PrettyPrint(gateway))
		//log.Println(utils.PrettyPrint(WebGatewayToTtnMapperGateway(*gateway)))
		log.Println(WebGatewayToTtnMapperGateway(*gateway).Description)
	}
}
