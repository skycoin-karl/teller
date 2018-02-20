package main

import (
	"net/http"

	"github.com/skycoin-karl/teller/dropper"
	"github.com/skycoin-karl/teller/model"
	"github.com/skycoin-karl/teller/monitor"
	"github.com/skycoin-karl/teller/scanner"
	"github.com/skycoin-karl/teller/sender"
	"github.com/skycoin-karl/teller/skycoin"
)

var (
	DROPPER *dropper.Dropper
	SKYCOIN *skycoin.Connection
	SCANNER *scanner.Scanner
	SENDER  *sender.Sender
	MONITOR *monitor.Monitor
	MODEL   *model.Model
)

func main() {
	var err error

	DROPPER, err = dropper.NewDropper()
	if err != nil {
		panic(err)
	}

	SKYCOIN, err = skycoin.NewConnection("localhost:6430", "seed")
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

	http.HandleFunc("/api/bind", apiBind)
	http.HandleFunc("/api/status", apiStatus)

	if err = http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
