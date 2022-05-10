package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// CSOContract contract for managing CRUD for EVs
type CSOContract struct {
	contractapi.Contract
}

// QueryResult structure used for handling result of query
type QueryResult struct {
	Record    *CSO
	TxId      string    `json:"txId"`
	Timestamp time.Time `json:"timestamp"`
}

// CreateCSOUser creates a new instance of cSO
func (c *CSOContract) CreateCSOUser(ctx contractapi.TransactionContextInterface, CSOID string, numChargers int) error {
	csoUser := new(CSO)
	csoUser.CSOID = CSOID
	exists, err := csoUser.LoadState(ctx)
	if err != nil {
		return fmt.Errorf("Could not read from world state. %s", err)
	} else if exists {
		return fmt.Errorf("The CSO %s already existss", CSOID)
	}

	newCSO := new(CSO)
	newCSO.CSOID = CSOID
	chargersSlice := make([]Charger, numChargers)
	for i := 1; i <= numChargers; i++ {
		newCharger := new(Charger)
		newCharger.ChargerID = i
		chargersSlice[i-1] = *newCharger
	}
	newCSO.Chargers = chargersSlice[:]
	return newCSO.SaveState(ctx)
}

// ReadCSOData retrieves an instance of CSO from the world state
func (c *CSOContract) ReadCSOData(ctx contractapi.TransactionContextInterface, CSOID string) (*CSO, error) {
	evUser := new(CSO)
	evUser.CSOID = CSOID
	exists, err := evUser.LoadState(ctx)
	if err != nil {
		return nil, fmt.Errorf("Could not read from world state. %s", err)
	} else if !exists {
		return nil, fmt.Errorf("The EV User %s does not exist", CSOID)
	}

	return evUser, nil
}

// DeleteCSOUser deletes an CSO from the world state
func (c *CSOContract) DeleteCSOUser(ctx contractapi.TransactionContextInterface, CSOID string) error {
	evUser := new(CSO)
	evUser.CSOID = CSOID
	exists, err := evUser.LoadState(ctx)
	if err != nil {
		return fmt.Errorf("Could not read from world state. %s", err)
	} else if !exists {
		return fmt.Errorf("The EV User %s does not exist.", CSOID)
	}

	return ctx.GetStub().DelState(CSOID)
}

// TransactEnergy retrieves an EV from the world state and updates its value
// Important: Make sure to configure channel name
func (c *CSOContract) TransactEnergy(ctx contractapi.TransactionContextInterface, CSOID string, EVID string, ChargerID int, PowerFlow float64, RecentMoney float64, Temperature float64, SoC float64, SoH float64) ([]byte, error) {
	channelName := "default-channel"

	csoUser := new(CSO)
	csoUser.CSOID = CSOID
	exists, err := csoUser.LoadState(ctx)
	if err != nil {
		return nil, fmt.Errorf("Could not read from world state. %s", err)
	} else if !exists {
		return nil, fmt.Errorf("The EV User %s does not exist", EVID)
	}
	if len(csoUser.Chargers) < ChargerID {
		return nil, fmt.Errorf("Invalid Charger ID %d", ChargerID)
	}

	// Send EV Struct TX Update
	invokeArgs := [][]byte{[]byte("UpdateEVData"), []byte(fmt.Sprint(EVID)), []byte(fmt.Sprint(CSOID)), []byte(fmt.Sprint(ChargerID)),
		[]byte(fmt.Sprint(PowerFlow)), []byte(fmt.Sprint(RecentMoney)), []byte(fmt.Sprint(Temperature)), []byte(fmt.Sprint(SoC)), []byte(fmt.Sprint(SoH))}
	response := ctx.GetStub().InvokeChaincode("EV", invokeArgs, channelName)
	if response.Status != 200 {
		return nil, fmt.Errorf("Error invoking EV Chaincode. %s", response.GetMessage())
	}

	// CSO Struct TX Update
	csoUser.Chargers[ChargerID-1].EVID = EVID
	csoUser.Chargers[ChargerID-1].PowerFlow = PowerFlow
	error := csoUser.SaveState(ctx)
	if error != nil {
		return nil, fmt.Errorf("Error saving state for CSOID %s", CSOID)
	}
	return response.GetPayload(), nil
}

// QueryAssetHistory returns the chain of custody for a asset since issuance
func (c *CSOContract) QueryAssetHistory(ctx contractapi.TransactionContextInterface, CSOID string) ([]QueryResult, error) {
	csoUser := new(CSO)
	csoUser.CSOID = CSOID
	compositeKey, err := csoUser.ToCompositeKey(ctx)
	resultsIterator, err := ctx.GetStub().GetHistoryForKey(compositeKey)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var results []QueryResult
	for resultsIterator.HasNext() {
		queryResult, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var cso *CSO
		err = json.Unmarshal(queryResult.Value, &cso)
		if err != nil {
			return nil, err
		}

		timestamp, err := ptypes.Timestamp(queryResult.Timestamp)
		if err != nil {
			return nil, err
		}
		record := QueryResult{
			TxId:      queryResult.TxId,
			Timestamp: timestamp,
			Record:    cso,
		}
		results = append(results, record)
	}

	return results, nil
}
