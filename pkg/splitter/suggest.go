package splitter

import (
	"log"
	"sort"
	"time"
)

func CreateTotalSuggestionWrapper(tripId int64, totalAmount float64, membersJson string, currentMemberEmail string, sharesJson string) *PlanSuggestion {

	members, err := parseMembers(membersJson)
	if err != nil {
		log.Println("Error while decoding membersJson ", err)
		return &PlanSuggestion{}
	}
	shares, err := parseShares(sharesJson)
	if err != nil {
		log.Println("Error while decoding sharesJson ", err)
		return &PlanSuggestion{}
	}

	return CreateTotalSuggestion(tripId, totalAmount, members, currentMemberEmail, shares)
}

func CreateIndividualSuggestionWrapper(tripId, planId int64, amount float64, notes string, created time.Time, membersJson string, currentMemberEmail string, sharesJson string) *PlanSuggestion {

	members, err := parseMembers(membersJson)
	if err != nil {
		log.Println("Error while decoding membersJson ", err)
		return &PlanSuggestion{}
	}

	shares, err := parseShares(sharesJson)
	if err != nil {
		log.Println("Error while decoding sharesJson ", err)
		return &PlanSuggestion{}
	}
	return createIndividualSuggestion(tripId, planId, amount, notes, created, members, currentMemberEmail, shares)
}

func CreateTotalSuggestion(tripId int64, totalAmount float64, members []Member, currentMemberEmail string, shares []Share) *PlanSuggestion {
	planSuggestion := &PlanSuggestion{
		Tripid:      tripId,
		Planid:      0,
		Notes:       "Total",
		Brief:       "",
		Date:        "--",
		Amount:      totalAmount,
		Operation:   OpNotInvolved,
		Suggestions: make([]Suggestion, 0),
	}
	//generateBenefactorSuggestions(0, members, allShares, planSuggestion)
	allShares, sharesPresentAlready := mergeDuplicateShares(tripId, 0, members, shares)
	// create/update shares from member name and avatar if not present already
	allShares = createShares(tripId, 0, members, allShares, sharesPresentAlready)
	posShares, negShares := posNegShares(0, allShares)
	generateSuggestions(posShares, negShares, planSuggestion)
	addCurrentUserBrief(planSuggestion, currentMemberEmail)
	return planSuggestion
}

func createIndividualSuggestion(tripId, planId int64, amount float64, notes string, created time.Time, members []Member, currentMemberEmail string, shares []Share) *PlanSuggestion {
	planSuggestion := &PlanSuggestion{
		Tripid:      tripId,
		Planid:      planId,
		Notes:       notes,
		Brief:       "",
		Date:        formatTimeMonth(created),
		Amount:      amount,
		Operation:   OpNotInvolved,
		Suggestions: make([]Suggestion, 0),
	}

	if amount > 0 {
		// should be called before merging suggestions, so that it will create suggestions for each and every shares
		generateBenefactorSuggestions(planId, members, shares, planSuggestion)
		// shares might have multiple values for the same memberid If he paid the amount in stages.
		allShares, sharesPresentAlready := mergeDuplicateShares(tripId, planId, members, shares)
		// create/update shares from member name and avatar if not present already
		allShares = createShares(tripId, planId, members, allShares, sharesPresentAlready)
		posShares, negShares := posNegShares(planId, allShares)
		generateSuggestions(posShares, negShares, planSuggestion)
		addCurrentUserBrief(planSuggestion, currentMemberEmail)
	}
	return planSuggestion
}

// filter positive and negative shares in a two different array
func posNegShares(planId int64, shares []Share) ([]*Share, []*Share) {
	posShares := make([]*Share, 0)
	negShares := make([]*Share, 0)
	for i := 0; i < len(shares); i++ {
		share := shares[i]

		// allow zero planid for total shares
		if planId == 0 || share.Planid == planId {
			share.Diff = share.Paid - share.Share
			if share.Diff >= 0 {
				posShares = append(posShares, &share)
			} else {
				negShares = append(negShares, &share)
			}
		}
	}

	return posShares, negShares
}

// GenerateBenefactor suggestions. Should be called before merging suggestions
func generateBenefactorSuggestions(planId int64, members []Member, shares []Share, planSuggestion *PlanSuggestion) {
	shares = populateMemberNameAndAvatar(members, shares)
	for _, share := range shares {
		if isCurrentPlan(share.Planid, planId) {
			if share.Paid > 0 {
				paidSuggestion := createBenefactorSuggestion(&share, members)
				if planSuggestion != nil {
					planSuggestion.Suggestions = append(planSuggestion.Suggestions, paidSuggestion)
				}
			}
		}
	}
}

func populateMemberNameAndAvatar(members []Member, shares []Share) []Share {
	for i := 0; i < len(shares); i++ {
		share := &shares[i]
		member := getMember(members, share.Memberemail)
		share.Membername = member.Name
		share.Memberavatar = member.Avatar
	}
	return shares
}

// Generate suggestions is a recursive funtion. It calls the same function if the positive share is greater than zero.
// It breaks on the event of share's positive diff == 0.
// Each call has to sort the shares again to suggest big amounts of the current set.
func generateSuggestions(posShares []*Share, negShares []*Share, planSuggestion *PlanSuggestion) {

	//sort suggestions - so that big payer should tally big defaulter
	sort.Slice(posShares, func(i, j int) bool { return posShares[i].Diff > posShares[j].Diff })
	sort.Slice(negShares, func(i, j int) bool { return negShares[i].Diff < negShares[j].Diff })

	for i := 0; i < len(posShares); i++ {
		positiveShare := posShares[i]
		//fmt.Printf("positiveShare %+v\n", *positiveShare)

		// Already settled
		if isTallyed(positiveShare.Diff) { // If need add this too.
			// suggestion := createSuggestion(0, positiveShare, nil)
			// planSuggestion.Suggestions = append(planSuggestion.Suggestions, suggestion)
			// log.Println("suggestion suggestion  ", suggestion)
			continue
		}

		for j := 0; j < len(negShares); j++ {
			negativeShare := negShares[j]
			//fmt.Printf("negativeShare %+v\n", *negativeShare)

			if isTallyed(negativeShare.Diff) {
				continue
			}

			if positiveShare.Diff >= -negativeShare.Diff {
				suggestion := createSuggestion(-negativeShare.Diff, positiveShare, negativeShare)
				planSuggestion.Suggestions = append(planSuggestion.Suggestions, suggestion)
				positiveShare.Diff = positiveShare.Diff + negativeShare.Diff
				negativeShare.Diff = 0
				//fmt.Printf("positive suggestion %+v\n", suggestion)
				generateSuggestions(posShares, negShares, planSuggestion)
				return
			} else {
				suggestion := createSuggestion(positiveShare.Diff, positiveShare, negativeShare)
				planSuggestion.Suggestions = append(planSuggestion.Suggestions, suggestion)
				negativeShare.Diff = negativeShare.Diff + positiveShare.Diff
				positiveShare.Diff = 0
				//fmt.Printf("negative suggestion %+v\n", suggestion)
				generateSuggestions(posShares, negShares, planSuggestion)
				return
			}

			if isTallyed(positiveShare.Diff) {
				break
			}
		}
	}

}

func createSuggestion(payableAmount float64, positiveShare *Share, negativeShare *Share) Suggestion {
	operation := OpGetsBack
	if payableAmount == 0 {
		operation = OpSettled
	}

	suggestion := Suggestion{}
	if positiveShare != nil {
		suggestion.AMemberemail = positiveShare.Memberemail
		suggestion.AMembername = positiveShare.Membername
		suggestion.AMemberavatar = positiveShare.Memberavatar
	}

	if negativeShare != nil {
		suggestion.BMemberemail = negativeShare.Memberemail
		suggestion.BMembername = negativeShare.Membername
		suggestion.BMemberavatar = negativeShare.Memberavatar
	}

	suggestion.Amount = payableAmount
	suggestion.Operation = operation
	return suggestion
}

func createBenefactorSuggestion(payer *Share, members []Member) Suggestion {
	suggestion := Suggestion{}
	suggestion.AMemberemail = payer.Memberemail
	suggestion.AMembername = payer.Membername
	suggestion.AMemberavatar = payer.Memberavatar
	suggestion.Datestr = formatTimeSmall(payer.Created)

	benefactor := getMember(members, payer.Benefactoremail)

	if payer.Memberemail != payer.Benefactoremail {
		suggestion.BMemberemail = payer.Benefactoremail
		suggestion.BMembername = benefactor.Name
		suggestion.BMemberavatar = benefactor.Avatar
	} else {
		suggestion.BMemberemail = ""
		suggestion.BMembername = "the bill"
		suggestion.BMemberavatar = ""
	}

	suggestion.Amount = payer.Paid
	suggestion.Operation = OpPaid
	return suggestion
}

func addCurrentUserBrief(planSuggestion *PlanSuggestion, currentMemberEmail string) {
	ownGetsBackSuggestions, ownGiveSuggestions := splitSuggestionsByGettersAndGivers(planSuggestion, currentMemberEmail)
	if len(ownGetsBackSuggestions) == 0 && len(ownGiveSuggestions) == 0 {
		planSuggestion.Brief = "Settled"
		planSuggestion.Operation = OpSettled
	} else {
		getsSentence, getsBackAmount := formatSentence(ownGetsBackSuggestions, true)
		givesSentence, givesAmount := formatSentence(ownGiveSuggestions, false)

		// Populate brief
		if getsBackAmount > 0 {
			planSuggestion.Operation = OpGetsBack
			planSuggestion.Brief = "You gets back " + float64Str(getsBackAmount) + " from " + getsSentence
		}

		if givesAmount > 0 {
			if planSuggestion.Brief != "" {
				planSuggestion.Brief = planSuggestion.Brief + " and "
			}
			planSuggestion.Brief = "You have to give " + float64Str(givesAmount) + " to " + givesSentence

			if planSuggestion.Operation == OpGetsBack {
				planSuggestion.Operation = OpBoth
			} else {
				planSuggestion.Operation = OpPaid
			}

		}
	}
}

func splitSuggestionsByGettersAndGivers(planSuggestion *PlanSuggestion, currentMemberEmail string) ([]Suggestion, []Suggestion) {
	ownGetsBackSuggestions := make([]Suggestion, 0)
	ownGiveSuggestions := make([]Suggestion, 0)
	for i := 0; i < len(planSuggestion.Suggestions); i++ {
		suggestion := planSuggestion.Suggestions[i]
		if suggestion.Operation == OpPaid { // don't include random payments. Benefactor suggestions dropped.
			continue
		}

		if suggestion.AMemberemail == currentMemberEmail {
			ownGetsBackSuggestions = append(ownGetsBackSuggestions, suggestion)
		} else if suggestion.BMemberemail == currentMemberEmail {
			ownGiveSuggestions = append(ownGiveSuggestions, suggestion)
		}
	}
	return ownGetsBackSuggestions, ownGiveSuggestions
}

func formatSentence(suggestions []Suggestion, findGivers bool) (string, float64) {
	var sentence string
	var totalAmount float64
	lenOfSuggestions := len(suggestions)
	for i := 0; i < lenOfSuggestions; i++ {
		suggestion := suggestions[i]
		theOtherMemberName := suggestion.AMembername
		if findGivers {
			theOtherMemberName = suggestion.BMembername
		}
		totalAmount = totalAmount + suggestion.Amount

		if i == lenOfSuggestions-1 {
			if sentence == "" {
				sentence = theOtherMemberName + ". "
			} else {
				sentence = sentence + " and " + theOtherMemberName + ". "
			}

		} else {
			sentence = sentence + theOtherMemberName + ", "
		}

	}
	return sentence, totalAmount
}
