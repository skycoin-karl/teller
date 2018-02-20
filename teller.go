package main

import (
	"time"

	"github.com/skycoin-karl/teller/dropper"
	"github.com/skycoin-karl/teller/model"
	"github.com/skycoin-karl/teller/monitor"
	"github.com/skycoin-karl/teller/scanner"
	"github.com/skycoin-karl/teller/sender"
	"github.com/skycoin-karl/teller/skycoin"
)

func main() {
	var (
		DROPPER *dropper.Dropper
		SKYCOIN *skycoin.Connection
		SCANNER *scanner.Scanner
		SENDER  *sender.Sender
		MONITOR *monitor.Monitor
		MODEL   *model.Model

		err error
	)

	DROPPER, err = dropper.NewDropper()
	if err != nil {
		panic(err)
	}

	SKYCOIN, err = skycoin.NewConnection("addr", "seed")
	if err != nil {
		panic(err)
	}

	SCANNER, err = scanner.NewScanner(DROPPER)
	if err != nil {
		panic(err)
	}

	SENDER, err = sender.NewSender(SKYCOIN, DROPPER)
	if err != nil {
		panic(err)
	}

	MONITOR, err = monitor.NewMonitor(SKYCOIN)
	if err != nil {
		panic(err)
	}

	MODEL, err = model.NewModel("db/", SCANNER, SENDER, MONITOR)
	if err != nil {
		panic(err)
	}

	_ = MODEL

	<-time.After(time.Second * 20)
}
