package monitor

import (
	"time"

	"github.com/skycoin-karl/teller/skycoin"
	"github.com/skycoin-karl/teller/types"
)

type Monitor struct {
	skycoin *skycoin.Connection
}

func NewMonitor(sky *skycoin.Connection) (*Monitor, error) {
	return &Monitor{sky}, nil
}

func (m *Monitor) Handle(r *types.Request) error {
	for !r.Metadata.Expired() {
		// wait before monitoring
		<-time.After(time.Minute)

		// get sky transaction
		tx, err := m.skycoin.Client.GetTransactionByID(r.Metadata.TxId)
		if err != nil {
			return err
		}

		// if not confirmed, check again
		if !tx.Transaction.Status.Confirmed {
			continue
		}

		// transaction confirmed, all done
		r.Metadata.Status = types.DONE
		return nil
	}

	// if loop breaks, then request has expired
	r.Metadata.Status = types.EXPIRED
	return nil
}
