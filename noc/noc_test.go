package noc

import (
	"log"
	"testing"
	"ttnmapper-gateway-update/utils"
)

func TestFetchNocStatuses(t *testing.T) {
	gateways, err := FetchNocStatuses()
	if err != nil {
		t.Fatalf(err.Error())
	}
	log.Println(utils.PrettyPrint(gateways))
}
