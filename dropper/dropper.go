package dropper

import (
	"errors"

	"github.com/skycoin-karl/teller/types"
)

type Dropper struct {
	Connections types.Connections
}

func NewDropper() (*Dropper, error) {
	btc, err := NewBTCConnection("localhost:8332")

	return &Dropper{
		Connections: types.Connections{types.BTC: btc},
	}, err
}

var ErrConnectionMissing = errors.New("connection doesn't exist")

func (d *Dropper) GetBalance(c types.Currency, a types.Drop) (float64, error) {
	connection, exists := d.Connections[c]
	if !exists {
		return 0.0, ErrConnectionMissing
	}

	return connection.Balance(a)
}

// GetValue returns SKY value of the amount of currency.
func (d *Dropper) GetValue(c types.Currency, amount float64) uint64 {
	return 0
}
