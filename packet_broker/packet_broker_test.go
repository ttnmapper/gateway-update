package packet_broker

import (
	"log"
	"testing"
	"ttnmapper-gateway-update/utils"
)

func TestFetchStatuses(t *testing.T) {
	gateways, err := FetchStatuses()
	if err != nil {
		t.Fatalf(err.Error())
	}

	for _, gateway := range gateways {
		log.Println(utils.PrettyPrint(gateway))
		ttnMapperGw, err := PbGatewayToTtnMapperGateway(gateway)
		if err != nil {
			t.Fatalf(err.Error())
		}
		log.Println(utils.PrettyPrint(ttnMapperGw))
	}
}
