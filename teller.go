package main

import (
	"net/http"
	"os"
	"os/signal"

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
	// for graceful shutdown / cleanup
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

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
	SCANNER.Start()

	SENDER, err = sender.NewSender(SKYCOIN, DROPPER)
	if err != nil {
		panic(err)
	}
	SENDER.Start()

	MONITOR, err = monitor.NewMonitor(SKYCOIN)
	if err != nil {
		panic(err)
	}
	MONITOR.Start()

	MODEL, err = model.NewModel("db/", SCANNER, SENDER, MONITOR)
	if err != nil {
		panic(err)
	}
	MODEL.Start()

	go func() {
		<-stop
		MODEL.Stop()
		os.Exit(0)
	}()

	http.HandleFunc("/api/bind", apiBind)
	http.HandleFunc("/api/status", apiStatus)

	println("listening on :8080")
	if err = http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
