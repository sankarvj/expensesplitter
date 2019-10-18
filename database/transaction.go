package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

//Trip ...
type Trip struct {
	Name         string
	Transactions []Transaction
}

//Transaction ...
type Transaction struct {
	Name   string
	Amount float64
	Shares []Share
}

//Share ...
type Share struct {
	Member string
	Amount float64
}

//NewTrip ...
func NewTrip(tripName, transactionName string, members []string, sharesSlice []float64) error {
	var shares []Share
	for i, member := range members {
		share := Share{
			Member: member,
			Amount: sharesSlice[i],
		}
		shares = append(shares, share)
	}

	transaction := Transaction{
		Name:   transactionName,
		Shares: shares,
	}

	trip := &Trip{}

	d := 24 * time.Hour
	key := time.Now().Truncate(d).Format(time.RFC3339)

	result, err := retriveData(tripName, key)
	if err != nil {
		switch err {
		case errBucketNotFound:
			trip = &Trip{
				Name:         tripName,
				Transactions: []Transaction{transaction},
			}
		default:
			return err
		}
	} else {
		fmt.Println("result --", result)
		if result != "" {
			err = json.Unmarshal([]byte(result), trip)
			if err != nil {
				return err
			}
		} else {
			trip = &Trip{
				Name:         tripName,
				Transactions: []Transaction{transaction},
			}
		}
		if err := deDoupTransactionName(trip.Transactions, transaction.Name); err != nil {
			return err
		}
		trip.Transactions = append(trip.Transactions, transaction)
	}

	return storeNewTransactionData(trip, key)
}

func storeNewTransactionData(trip *Trip, key string) error {
	json, err := json.Marshal(trip)
	if err != nil {
		return err
	}
	return storeData(trip.Name, key, json)
}

func deDoupTransactionName(transactions []Transaction, transactionName string) error {
	for _, transaction := range transactions {
		if transaction.Name == transactionName {
			return errors.New("Could not have duplicate transaction on the same day")
		}
	}
	return nil
}
