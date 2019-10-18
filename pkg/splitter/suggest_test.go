package splitter

import (
	"fmt"
	"strconv"
	"testing"
)

func TestCreateTotalSuggestion(t *testing.T) {
	members := createDummyMembers()
	shares := createDummyShares()

	fmt.Println("members ......", members)
	fmt.Println("shares ......", shares)

	suggestions := CreateTotalSuggestion(1, 4000, members, "vijay_0", shares)
	fmt.Println("suggestions ......", suggestions)
}

func createDummyMembers() []Member {
	members := make([]Member, 3)

	for i := 0; i < 3; i++ {
		member := Member{
			Tripid: 1,
			Email:  "vijay_" + strconv.FormatInt(int64(i), 10),
			Name:   "vijay_" + strconv.FormatInt(int64(i), 10),
		}
		members[i] = member
	}
	return members
}

func createDummyShares() []Share {
	shares := make([]Share, 3)

	for i := 0; i < 3; i++ {
		share := Share{
			Tripid:      1,
			Memberemail: "vijay_" + strconv.FormatInt(int64(i), 10),
			Membername:  "vijay_" + strconv.FormatInt(int64(i), 10),
			Share:       1000,
			Paid:        100,
		}
		shares[i] = share
	}
	return shares
}
