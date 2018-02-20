package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/skycoin-karl/teller/types"
	"github.com/skycoin/skycoin/src/cipher"
)

type apiBindRequest struct {
	Address      string `json:"address"`
	DropCurrency string `json:"drop_currency"`
}

type apiBindResponse struct {
	DropAddress  string `json:"drop_address"`
	DropCurrency string `json:"drop_type"`
}

func apiBind(w http.ResponseWriter, r *http.Request) {
	req := apiBindRequest{}

	// decode json
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// decode drop_currency
	currency := types.Currency(req.DropCurrency)

	// decode skycoin address
	address, err := cipher.DecodeBase58Address(req.Address)
	if err != nil {
		http.Error(w, "invalid skycoin address", http.StatusBadRequest)
		return
	}

	// generate drop address
	drop, err := DROPPER.Connections[currency].Generate()
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	request := &types.Request{
		Address:  types.Address(address.String()),
		Currency: currency,
		Drop:     drop,
		Metadata: &types.Metadata{
			Status:    types.DEPOSIT,
			CreatedAt: time.Now().Unix(),
			UpdatedAt: time.Now().Unix(),
			TxId:      "",
		},
	}

	// send json response
	err = json.NewEncoder(w).Encode(&apiBindResponse{
		DropAddress:  string(request.Drop),
		DropCurrency: string(request.Currency),
	})
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	// TODO: handle error better
	err = MODEL.Handle(request)
	if err != nil {
		panic(err)
	}
}

type apiStatusRequest struct {
	Address      string `json:"skycoin_address"`
	DropAddress  string `json:"drop_address"`
	DropCurrency string `json:"drop_currency"`
}

type apiStatusResponse struct {
	Status    string `json:"status"`
	UpdatedAt int64  `json:"updated_at"`
}

func apiStatus(w http.ResponseWriter, r *http.Request) {
	req := apiStatusRequest{}

	// decode json
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// decode skycoin address
	address, err := cipher.DecodeBase58Address(req.Address)
	if err != nil {
		http.Error(w, "invalid skycoin_address", http.StatusBadRequest)
		return
	}

	meta, err := MODEL.GetMetadata(
		types.Address(address.String()),
		types.Drop(req.DropAddress),
		types.Currency(req.DropCurrency),
	)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(&apiStatusResponse{
		Status:    string(meta.Status),
		UpdatedAt: meta.UpdatedAt,
	})
}
