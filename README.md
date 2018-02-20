# teller

<p align="center">
	<img src="img/overview.svg" />
</p>

## scanner

<p align="center">
	<img src="img/scanner.svg" />
</p>

The **scanner** continuously checks the balance of `request.Drop` and returns when the balance is greater than zero (user made a deposit), or the request has expired (hours/days with no activity).

### possible errors

* Getting balance of `request.Drop` from multicoin wallet service.

## sender

<p align="center">
	<img src="img/sender.svg" />
</p>

The **sender** handles a request once a deposit has been detected. It calculates the equivalent skycoin amount to send based on the current value of `request.Currency`. The **sender** then gets skycoin from an exchange (using exchange api service), or from the OTC wallet. Then, a skycoin transaction is created (using multicoin wallet api) and injected (using multicoin wallet api). The `request.TxId` is filled with the skycoin transaction ID for **monitor** to watch.

### possible errors

* Getting skycoin from exchange api service.
* Creating skycoin transaction from multicoin wallet service.
* Injecting skycoin transaction from multicoin wallet service.

## monitor

<p align="center">
	<img src="img/monitor.svg" />
</p>