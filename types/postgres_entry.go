package types

import (
	"time"
)

type Packet struct {
	ID   uint
	Time time.Time `gorm:"not null;index=time"`

	DeviceID uint `gorm:"not null;index=device"`

	FPort uint8
	FCnt  uint32

	FrequencyID  uint
	DataRateID   uint
	CodingRateID uint

	// Gateway data
	GatewayID              uint `gorm:"not null;index=gateway"`
	GatewayTime            *time.Time
	Timestamp              *uint32
	FineTimestamp          *uint64
	FineTimestampEncrypted *[]byte
	FineTimestampKeyID     *uint
	ChannelIndex           uint32
	Rssi                   float32  `gorm:"type:numeric(6,2)"`
	SignalRssi             *float32 `gorm:"type:numeric(6,2)"`
	Snr                    *float32 `gorm:"type:numeric(5,2)"`

	Latitude         float64  `gorm:"not null;type:numeric(10,6);index:latitude"`
	Longitude        float64  `gorm:"not null;type:numeric(10,6);index:longitude"`
	Altitude         float64  `gorm:"type:numeric(6,1)"`
	AccuracyMeters   *float64 `gorm:"type:numeric(6,2)"`
	Satellites       *int32
	Hdop             *float64 `gorm:"type:numeric(4,1)"`
	AccuracySourceID uint

	ExperimentID *uint

	UserID      uint
	UserAgentID uint

	DeletedAt *time.Time
}

type Device struct {
	ID      uint
	AppId   string `gorm:"UNIQUE_INDEX:app_device"`
	DevId   string `gorm:"UNIQUE_INDEX:app_device"`
	DevEui  string `gorm:"UNIQUE_INDEX:app_device"`
	Packets []Packet
}

type Frequency struct {
	ID      uint
	Herz    uint64 `gorm:"unique;not null"`
	Packets []Packet
}

type DataRate struct {
	ID              uint
	Modulation      string `gorm:"UNIQUE_INDEX:data_rate"` // LORA or FSK or LORA-E
	Bandwidth       uint64 `gorm:"UNIQUE_INDEX:data_rate"`
	SpreadingFactor uint8  `gorm:"UNIQUE_INDEX:data_rate"`
	Bitrate         uint64 `gorm:"UNIQUE_INDEX:data_rate"`
	Packets         []Packet
}

type CodingRate struct {
	ID      uint
	Name    string `gorm:"unique;not null"`
	Packets []Packet
}

type AccuracySource struct {
	ID      uint
	Name    string `gorm:"unique;not null"`
	Packets []Packet
}

type Experiment struct {
	ID      uint
	Name    string `gorm:"unique;not null"`
	Packets []Packet
}

type User struct {
	ID         uint
	Identifier string `gorm:"unique;not null"`
	Packets    []Packet
}

type UserAgent struct {
	ID      uint
	Name    string `gorm:"unique;not null"`
	Packets []Packet
}

type Gateway struct {
	ID           uint
	GtwId        string `gorm:"UNIQUE_INDEX:gtw_id_eui_antenna"`
	GtwEui       string `gorm:"UNIQUE_INDEX:gtw_id_eui_antenna"`
	AntennaIndex uint8  `gorm:"UNIQUE_INDEX:gtw_id_eui_antenna"`

	Latitude         *float64
	Longitude        *float64
	Altitude         *int32
	LocationAccuracy *int32
	LocationSource   *string

	//AtLocationSince	time.Time // This value gets updated when the gateway moves
	LastHeard time.Time // This value always gets updated to reflect that the gateway is working

	Packets          []Packet
	GatewayLocations []GatewayLocation
}

type GatewayBlacklist struct {
	ID     uint
	GtwId  string `gorm:"index=idx_gateway_id"`
	GtwEui string `gorm:"index=idx_gateway_eui"`
}

type GatewayLocation struct {
	ID        uint
	GatewayID uint `gorm:"not null;index=idx_gateway_id"`

	InstalledAt time.Time `gorm:"index=idx_installed_at"`
	Latitude    float64
	Longitude   float64
}

type GatewayLocationForce struct {
	ID        uint
	GatewayID uint `gorm:"unique;not null"`

	Latitude  float64
	Longitude float64
}

type FineTimestampKeyID struct {
	ID                          uint
	FineTimestampEncryptedKeyId string
}

// Indexers: These structs are the same as the ones above, but used to index the cache maps
type DeviceIndexer struct {
	DevId  string
	AppId  string
	DevEui string
}

type GatewayIndexer struct {
	GtwId        string
	GtwEui       string
	AntennaIndex uint8
}

type DataRateIndexer struct {
	Modulation      string // LORA or FSK or LORA-E
	Bandwidth       uint64
	SpreadingFactor uint8
	Bitrate         uint64
}
