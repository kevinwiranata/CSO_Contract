package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// CSOObjectType for composite key
const CSOObjectType = "CS-Operator"

type Charger struct {
	EVID      string  `json:"EVID"` // null if no EV connected
	ChargerID int     `json:"charger_ID"`
	PowerFlow float64 `json:"power_flow"` // must be same as EV powerflow
}

type CSO struct {
	CSOID          string    `json:"ID"`
	TotalPowerFlow float64   `json:"total_power_flow"`
	EVCount        int       `json:"ev_count"`
	Chargers       []Charger `json:"charger"`
}

// ToCompositeKey returns a composite key based on the ID and accountType
func (c *CSO) ToCompositeKey(ctx contractapi.TransactionContextInterface) (string,
	error) {
	attributes := []string{
		c.CSOID,
	}
	return ctx.GetStub().CreateCompositeKey(CSOObjectType, attributes)
}

// ToLedgerValue creates a JSON-encoded account
func (c *CSO) ToLedgerValue() ([]byte, error) {
	return json.Marshal(c)
}

// SaveState saves the accounts into the ledger
func (c *CSO) SaveState(ctx contractapi.TransactionContextInterface) error {
	compositeKey, err := c.ToCompositeKey(ctx)
	if err != nil {
		message := fmt.Sprintf("Unable to create a composite key: %s", err.Error())
		return errors.New(message)
	}

	ledgerValue, err := c.ToLedgerValue()

	if err != nil {
		message := fmt.Sprintf("Unable to  compose a ledger value: %s", err.Error())
		return errors.New(message)
	}
	return ctx.GetStub().PutState(compositeKey, ledgerValue)
}

// LoadState loads the data from the ledger into the EV object if the data is found
// Returns false if an Account object wasn't found in the ledger; otherwise
//returns true.
func (c *CSO) LoadState(ctx contractapi.TransactionContextInterface) (bool, error) {
	compositeKey, err := c.ToCompositeKey(ctx)
	if err != nil {
		message := fmt.Sprintf("Unable to create a composite key: %s", err.Error())
		return false, errors.New(message)
	}

	ledgerValue, err := ctx.GetStub().GetState(compositeKey)
	if err != nil {
		message := fmt.Sprintf("Unable to  compose a ledger value: %s", err.Error())
		return false, errors.New(message)
	}

	if ledgerValue == nil {
		return false, nil
	}

	return true, json.Unmarshal(ledgerValue, &c)
}
