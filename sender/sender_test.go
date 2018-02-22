package sender

import (
	"flag"
	"testing"

	"github.com/skycoin-karl/teller/dropper"
	"github.com/skycoin-karl/teller/skycoin"
	//"github.com/skycoin-karl/teller/types"
)

const (
	SKYCOIN_NODE = "localhost:6430"
)

var (
	SKYCOIN *skycoin.Connection
	DROPPER *dropper.Dropper

	SKYCOIN_SEED = flag.String(
		"skycoin_seed",
		"",
		"seed used for skycoin wallet daemon (sending coins)",
	)
)

func init() {
	flag.Parse()

	var err error

	SKYCOIN, err = skycoin.NewConnection(SKYCOIN_NODE, *SKYCOIN_SEED)
	if err != nil {
		panic(err)
	}

	DROPPER, err = dropper.NewDropper()
	if err != nil {
		panic(err)
	}
}

func TestNewSender(t *testing.T) {
	s, err := NewSender(SKYCOIN, DROPPER)
	if err != nil {
		panic(err)
	}

	if s == nil {
		t.Fatal("nil sender")
	}
}
