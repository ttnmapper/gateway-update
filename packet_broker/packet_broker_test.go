package packet_broker

import (
	"log"
	"testing"
)

func TestFetchStatusesPage(t *testing.T) {
	gateways, err := FetchStatuses(0)
	if err != nil {
		t.Fatalf(err.Error())
	}

	//for _, gateway := range gateways {
	//	log.Println(utils.PrettyPrint(gateway))
	//	ttnMapperGw, err := PbGatewayToTtnMapperGateway(gateway)
	//	if err != nil {
	//		t.Fatalf(err.Error())
	//	}
	//	log.Println(utils.PrettyPrint(ttnMapperGw))
	//}

	log.Printf("Fetched %d gateways", len(gateways))
}

func TestFetchStatusesAll(t *testing.T) {
	page := 0
	for {
		gateways, err := FetchStatuses(page)
		if err != nil {
			log.Println(err.Error())
			break
		} else {
			log.Printf("Fetched %d statuses for page %d", len(gateways), page)
		}
		page++
	}
}
