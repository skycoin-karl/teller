package monitor

import (
	"testing"

	"github.com/skycoin-karl/teller/skycoin"
	"github.com/skycoin-karl/teller/types"
)

const (
	SKYCOIN_NODE = "localhost:6430"
	SKYCOIN_SEED = "doesn't matter"
	SKYCOIN_TXID = "8d34b138f6a1da3330a798957215b89cae906a42fa35d2da8ed729d2c9a36ba8"
)

var (
	SKYCOIN *skycoin.Connection
)

func init() {
	var err error

	SKYCOIN, err = skycoin.NewConnection(SKYCOIN_NODE, SKYCOIN_SEED)
	if err != nil {
		panic(err)
	}
}

func TestNewMonitor(t *testing.T) {
	m, err := NewMonitor(SKYCOIN)
	if err != nil {
		t.Fatal(err)
	}

	if m == nil {
		t.Fatal("nil monitor")
	}
}

func TestStartStop(t *testing.T) {
	m, err := NewMonitor(SKYCOIN)
	if err != nil {
		t.Fatal(err)
	}

	m.Start()
	m.Stop()

	if !m.stopped {
		t.Fatal("monitor didn't stop")
	}
}

func TestProcessGoodRequest(t *testing.T) {
	m, err := NewMonitor(SKYCOIN)
	if err != nil {
		t.Fatal(err)
	}

	r := m.Handle(&types.Request{
		Address:  types.Address("doesn't matter"),
		Currency: types.Currency("doesn't matter"),
		Drop:     types.Drop("doesn't matter"),
		Metadata: &types.Metadata{
			Status:    types.CONFIRM,
			CreatedAt: 0,
			UpdatedAt: 0,
			TxId:      SKYCOIN_TXID,
		},
	})

	if m.work.Len() != 1 {
		t.Fatal("monitor.Handle() not queueing requests")
	}

	// only tick once
	go m.process()

	// wait for result
	result := <-r

	if result.Err != nil {
		t.Fatal(result.Err)
	}

	if result.Request.Metadata.Status != types.DONE {
		t.Fatal("monitor didn't correctly process request")
	}

	if m.work.Len() != 0 {
		t.Fatal("monitor.process() not dequeueing work")
	}
}

func TestProcessBadRequest(t *testing.T) {
	m, err := NewMonitor(SKYCOIN)
	if err != nil {
		t.Fatal(err)
	}

	r := m.Handle(&types.Request{
		Address:  types.Address("doesn't matter"),
		Currency: types.Currency("doesn't matter"),
		Drop:     types.Drop("doesn't matter"),
		Metadata: &types.Metadata{
			Status:    types.CONFIRM,
			CreatedAt: 0,
			UpdatedAt: 0,
			TxId:      "bad",
		},
	})

	// only tick once
	go m.process()

	// wait for result
	result := <-r

	if result.Err == nil {
		t.Fatal("monitor not returning request error")
	}

	if result.Request.Metadata.Status == types.DONE {
		t.Fatal("monitor processed bad request")
	}
}

func TestProcessTick(t *testing.T) {
	m, err := NewMonitor(SKYCOIN)
	if err != nil {
		t.Fatal(err)
	}

	r := m.Handle(&types.Request{
		Address:  types.Address("doesn't matter"),
		Currency: types.Currency("doesn't matter"),
		Drop:     types.Drop("doesn't matter"),
		Metadata: &types.Metadata{
			Status:    types.CONFIRM,
			CreatedAt: 0,
			UpdatedAt: 0,
			TxId:      SKYCOIN_TXID,
		},
	})

	m.Start()

	<-r

	m.Stop()
}

func TestQueueing(t *testing.T) {
	m, err := NewMonitor(SKYCOIN)
	if err != nil {
		t.Fatal(err)
	}

	results := make([]chan *types.Result, 2, 2)
	for i := range results {
		results[i] = m.Handle(&types.Request{
			Address:  types.Address("doesn't matter"),
			Currency: types.Currency("doesn't matter"),
			Drop:     types.Drop("doesn't matter"),
			Metadata: &types.Metadata{
				Status:    types.CONFIRM,
				CreatedAt: 0,
				UpdatedAt: 0,
				TxId:      SKYCOIN_TXID,
			},
		})
	}

	m.process()

	select {
	case <-results[0]:
		break
	default:
		t.Fatal("monitor processing work out of order")
	}
}
