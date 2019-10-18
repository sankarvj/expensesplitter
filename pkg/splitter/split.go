package splitter

import (
	"encoding/json"
	"log"
	"math"
	"strconv"
	"time"
)

//Member is the user who involved in the expense
type Member struct {
	Tripid  int64
	Name    string
	Email   string
	Avatar  string
	Created time.Time
	Updated int64
}

// Share the member has to pay
type Share struct {
	Id              int64
	Tripid          int64
	Planid          int64
	Memberemail     string
	Membername      string
	Memberavatar    string
	Benefactoremail string //The amount paid by this Memberid to this Benefactorid.
	Note            string
	Paid            float64 //Amount paid by this member.
	Share           float64 //Actual share he has to pay.
	Diff            float64 //Used internally to create suggestions
	Auto            bool
	Created         time.Time
	Updated         int64
}

//PlanSuggestion is the wrapper above suggestions has all the suggestions/settlements
type PlanSuggestion struct {
	Tripid      int64
	Planid      int64
	Notes       string
	Brief       string
	Date        string
	Amount      float64
	Operation   int
	Suggestions []Suggestion
}

//Suggestion is the one we can use to settle/show the expense
type Suggestion struct {
	AMemberemail  string
	AMembername   string
	AMemberavatar string
	BMemberemail  string
	BMembername   string
	BMemberavatar string
	Amount        float64
	Operation     int
	Datestr       string
}

const (
	OpNotInvolved = -1
	OpSettled     = 0
	OpGetsBack    = 1
	OpOwe         = 2 //not needed
	OpPaid        = 3
	OpBoth        = 4
)

func SplitSharesForBillWrapper(tripId int64, planId int64, billAmount float64, currentMemberEmail string, membersJson string, sharesJson string) (string, float64, bool) {

	members, err := parseMembers(membersJson)
	if err != nil {
		log.Println("Error while decoding membersJson ", err)
		return "", 0, false
	}
	shares, err := parseShares(sharesJson)
	if err != nil {
		log.Println("Error while decoding sharesJson ", err)
		return "", 0, false
	}

	allShares, meanShare, isEquallySplit := splitSharesForBill(tripId, planId, billAmount, currentMemberEmail, members, shares)
	allSharesJson, _ := json.Marshal(allShares)
	return string(allSharesJson), meanShare, isEquallySplit
}

// Calculate the share for each member in the group for the specific bill paid.
// tripId - The trip in which the expense made.
// planId - The plan for which the bill added.
// currentMemberId - Member who is adding/editing this bill.
// members - List of members involved in the bill
// shares - Already calculated share list. Empty otherwise. Share item inside share has a field called isFresh,
// make sure to mark that field false if you changes the share manually
// billAmount - Total bill amount for the expense made.
func splitSharesForBill(tripId int64, planId int64, billAmount float64, currentMemberEmail string, members []Member, shares []Share) ([]Share, float64, bool) {
	// shares might have multiple values for the same memberid If he paid the amount in stages.
	allShares, sharesPresentAlready := mergeDuplicateShares(tripId, planId, members, shares)
	// create/update shares from member name and avatar if not present already
	allShares = createShares(tripId, planId, members, allShares, sharesPresentAlready)
	// make the members who are not auto in the shares split equally.
	allShares, meanShare, isEquallySplit := splitUp(currentMemberEmail, allShares, billAmount)

	return allShares, meanShare, isEquallySplit
}

// Shares might have multiple values for the same memberid If he paid the amount in stages.
// This function will combine those shares for each member.
func mergeDuplicateShares(tripId int64, planId int64, members []Member, sharesWithDuplicates []Share) ([]Share, bool) {

	// creating new shares out of benefactor
	shares := createSharesOutOfBenefactor(planId, members, sharesWithDuplicates)

	//merging shares by memberid
	memberMap := make(map[string]*Share)
	for i := 0; i < len(shares); i++ {
		share := shares[i]
		// allow zero planid for calculating total shares
		if isCurrentPlan(share.Planid, planId) {
			if savedShare, ok := memberMap[share.Memberemail]; ok {
				if savedShare.Id == 0 { // dynamic share created from createSharesOutOfBenefactor
					savedShare.Id = share.Id
					savedShare.Membername = share.Membername
					savedShare.Memberavatar = share.Memberavatar
				}
				savedShare.Paid = savedShare.Paid + share.Paid
				savedShare.Share = savedShare.Share + share.Share
			} else {
				memberMap[share.Memberemail] = &share
			}
		}
	}

	// In go there is no way get set of values from map.
	allShares := make([]Share, 0)
	// find whether there are shares already present in the db
	sharesPresentAlready := false
	for _, share := range memberMap {
		if share.Id != 0 {
			sharesPresentAlready = true
		}
		allShares = append(allShares, *share)
	}
	return allShares, sharesPresentAlready
}

// Create new shares out of the benefactorid.
// The problem we are trying to solve is: Since the amount x is paid by member to a benefactor. A share created to that member with
// paid = -amount. So that the amount he paid for that bill amount will get tallyed.
func createSharesOutOfBenefactor(planId int64, members []Member, shares []Share) []Share {
	lengthOfShares := len(shares)
	for i := 0; i < lengthOfShares; i++ {
		share := shares[i]
		if share.Benefactoremail != share.Memberemail {
			benefactorShare := Share{
				Memberemail:     share.Benefactoremail,
				Membername:      "",
				Memberavatar:    "",
				Tripid:          share.Tripid,
				Planid:          share.Planid,
				Paid:            -share.Paid,
				Created:         share.Created,
				Benefactoremail: share.Benefactoremail,
			}
			shares = append(shares, benefactorShare)
		}
	}
	return shares
}

// create/update shares from member name and avatar if not present already
func createShares(tripId int64, planId int64, members []Member, shares []Share, sharesPresentAlready bool) []Share {

	for i := 0; i < len(members); i++ {
		member := members[i]
		share := getShare(member.Email, shares)

		if share == nil { //creae new share
			share = initShare(tripId, planId, member.Email, member.Name, member.Avatar, sharesPresentAlready)
			shares = append(shares, *share)
		} else { // share already available for this member
			share.Membername = member.Name
			share.Memberavatar = member.Avatar
		}
	}

	return shares
}

// Splitup has to handle cases such as:
// 1) For the first time, it has to update the share for sharers with the meanshare
// and update the current member as the default payer.
// 1a) For the first time, while updating the share for sharers with the meanshare, have to
// differentiate the payer who manually opt out of the share
// 2) If the share paid by anyone/multiple user/users changes, needs manual correction
// 3) If the share of the anyone/multiple user/users changes, others shares will be modified only if the auto is true
// 4) If the bill amount changes, needs manual correction
// 5) If new members added after the original bill, he will not be added to the old bill unless manual changes.
func splitUp(currentMemberEmail string, shares []Share, billAmount float64) ([]Share, float64, bool) {
	var totalAmountPaid float64 = 0
	isEquallySplit := true
	memberCount := len(shares)
	meanShare := billAmount / float64(memberCount)

	// before calculation remove manually added shares from the splitup calculation
	for i := 0; i < len(shares); i++ {
		share := &shares[i]
		if !share.Auto { // which means, he has calibrated/ manually changes the values. Don't include him in the meanshare calc.
			billAmount = billAmount - share.Share
			memberCount--
		}
		// calculating the total shares so that it can be used to compare with billamout and if the
		// mismatch is found, it will make all the shares null.
		totalAmountPaid = totalAmountPaid + share.Paid
	}

	// calculate autoShare
	if memberCount > 0 {
		meanShare = billAmount / float64(memberCount)
	}

	// loop again to set the meanshare
	for i := 0; i < len(shares); i++ {
		share := &shares[i]

		if share.Auto { // which means, he has been added automatically by looping members
			share.Share = meanShare

			// make current user pay by default
			if share.Memberemail == currentMemberEmail && share.Paid == 0 {
				share.Paid = billAmount
			}
		}

		if preciselyTwo(share.Share) != preciselyTwo(meanShare) {
			isEquallySplit = false
		}
	}

	return shares, meanShare, isEquallySplit
}

func ValidateShares(sharesJson string, billAmount float64) (bool, string) {
	shares, err := parseShares(sharesJson)
	if err != nil {
		log.Println("Error while decoding sharesJson ", err)
		return false, "Error while decoding sharesJson"
	}

	totalAmountPaid := 0.0
	totalShare := 0.0
	for _, share := range shares {
		totalAmountPaid = totalAmountPaid + share.Paid
		totalShare = totalShare + share.Share
	}

	totalAmountPaid = preciselyTwo(totalAmountPaid)
	totalShare = preciselyTwo(totalShare)
	billAmount = preciselyTwo(billAmount)

	// check if total amount paid is in par with the expense
	if totalAmountPaid > billAmount {
		return false, "Total amount paid more than the bill amount"
	} else if totalAmountPaid < billAmount {
		return false, "Total amount paid less than the bill amount"
	}

	// check if total amount shared is in par with the expense
	if totalShare > billAmount {
		return false, "Total share is more than the bill amount"
	} else if totalShare < billAmount {
		return false, "Total share is less than the bill amount"
	}

	return true, ""
}

func getMember(members []Member, memberEmail string) Member {
	for _, member := range members {
		if member.Email == memberEmail {
			return member
		}
	}
	return Member{}
}

// get share for the provided member id
func getShare(memberEmail string, shares []Share) *Share {
	for i := 0; i < len(shares); i++ {
		if shares[i].Memberemail == memberEmail {
			return &shares[i]
		}
	}
	return nil
}

func initShare(tripId int64, planId int64, memberEmail string, memberName string, memberAvatar string, sharesPresentAlready bool) *Share {
	share := Share{}
	share.Tripid = tripId
	share.Planid = planId
	share.Memberemail = memberEmail
	share.Membername = memberName
	share.Memberavatar = memberAvatar
	share.Benefactoremail = memberEmail
	share.Share = 0 //to differentiate between manual zero share with auto generated shares
	share.Paid = 0  //to differentiate between manual zero paid with auto generated shares
	share.Auto = true

	// Mark auto false if there are shares already present and this shares added due to new members addition after that.
	if sharesPresentAlready {
		share.Auto = false
	}

	return &share
}

const (
	smallLayout = "Jan 2 3:04PM"
	monthLayout = "Jan 2 2006"
)

func formatTimeSmall(t time.Time) string {
	return t.Format(smallLayout)
}

func formatTimeMonth(t time.Time) string {
	return t.Format(monthLayout)
}

func preciselyTwo(num float64) float64 {
	precision := 2
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}

func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func isTallyed(someValue float64) bool {
	if preciselyTwo(someValue) == 0 {
		return true
	}
	return false
}

func isCurrentPlan(planId, currentPlanId int64) bool {
	if planId == currentPlanId || currentPlanId == 0 {
		return true
	}
	return false
}

func float64Str(id float64) string {
	return strconv.FormatFloat(id, 'f', 2, 64)
}

func parseMembers(response string) ([]Member, error) {
	var obj []Member
	err := json.Unmarshal([]byte(response), &obj)
	return obj, err
}

func parseShares(response string) ([]Share, error) {
	var obj []Share
	err := json.Unmarshal([]byte(response), &obj)
	return obj, err
}
