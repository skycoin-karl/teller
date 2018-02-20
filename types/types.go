package types

import "time"

const (
	BTC Currency = "BTC"
	ETH Currency = "ETH"

	DEPOSIT Status = "waiting_deposit"
	SEND    Status = "waiting_send"
	CONFIRM Status = "waiting_confirm"
	DONE    Status = "done"
	EXPIRED Status = "expired"
)

type (
	Address  string
	Drop     string
	Currency string
	Status   string

	Metadata struct {
		Status    Status `json:"status"`
		CreatedAt int64  `json:"created_at"`
		UpdatedAt int64  `json:"updated_at"`
		TxId      string `json:"tx_id"`
	}

	Request struct {
		Address  Address
		Currency Currency
		Drop     Drop
		Metadata *Metadata
	}

	Service interface {
		Handle(*Request) error
	}

	Connection interface {
		Generate() (Drop, error)
		Balance(Drop) (float64, error)
		Connected() (bool, error)
		Stop() error
	}

	Connections map[Currency]Connection
)

func (m *Metadata) Update() { m.UpdatedAt = time.Now().Unix() }

func (m *Metadata) Expired() bool {
	return time.Since(time.Unix(m.UpdatedAt, 0)) > (time.Hour * 2)
}
