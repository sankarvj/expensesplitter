package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/sankarvj/expensesplitter/database"
	"github.com/urfave/cli"
)

func transactionFlags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:  "name, n",
			Value: "",
			Usage: "Name of the transaction (Required)",
		},
		cli.StringFlag{
			Name:  "members, m",
			Value: "",
			Usage: "Comma seperated names eg. gus, walt, jesse etc. (Required)",
		},
		cli.StringFlag{
			Name:  "share, s",
			Value: "",
			Usage: "Comma seperated shares in the same order of members eg. 100, 50, 500 etc. or leave it blank if its is shared equally. (Optional if expense provided)",
		},
		cli.StringFlag{
			Name:  "expense, e",
			Value: "",
			Usage: "Any valid amount (Optional if share provided)",
		},
		cli.BoolFlag{
			Name:  "delete, d",
			Usage: "Delete everything",
		},
	}
}

func suggestFlags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:  "today, t",
			Value: "",
			Usage: "dev,playground, staging or prod",
		},
	}
}

//TransactionCmd used to create/delete transaction
func TransactionCmd() cli.Command {
	return cli.Command{
		Name:  "transaction",
		Usage: "Adds new transaction",
		Flags: transactionFlags(),
		// the action, or code that will be executed when
		// we execute our `ns` command
		Action: func(c *cli.Context) error {
			// a simple lookup function
			transactionName := c.String("name")
			members := c.String("members")
			expense := c.String("expense")
			share := c.String("share")
			delete := c.Bool("delete")

			if delete {
				_, yes := waitforinput("Do you really want to delete everything? (yes/no)")
				if yes {
					database.DeleteBucket("default")
				}
				return nil
			}

			if transactionName == "" {
				fmt.Printf("%s  Please give the transaction name\n", devil())
				return nil
			}

			if members == "" {
				fmt.Printf("%s  Please give atleast one member name\n", devil())
				return nil
			}

			membersSlice := strings.Split(members, ",")
			shareSlice := make([]float64, len(membersSlice))

			if share == "" {
				if expense == "" {
					fmt.Printf("%s  Please provide either share or total expense. \n", devil())
					return nil
				}
				expenseInteger, err := strconv.ParseFloat(expense, 64)
				if err != nil {
					fmt.Printf("%s  Please enter valid expense\n", devil())
					return nil
				}
				totalMembers := len(membersSlice)
				share := expenseInteger / float64(totalMembers)
				for i := range membersSlice {
					shareSlice[i] = share
				}
			} else {
				shares := strings.Split(share, ",")

				if len(shares) != len(membersSlice) {
					fmt.Printf("%s  Given members and their shares are not matching\n", devil())
					return nil
				}

				for i := range membersSlice {
					eachShare := shares[i]
					eachShareInteger, err := strconv.ParseFloat(eachShare, 64)
					if err != nil {
						fmt.Printf("%s  Please enter share amount\n", devil())
						return nil
					}
					shareSlice[i] = eachShareInteger
				}
			}

			time.Sleep(1 * time.Second)
			err := database.NewTrip("default", transactionName, membersSlice, shareSlice)
			if err != nil {
				fmt.Printf("%s  %s\n", devil(), err.Error())
				return nil
			}
			fmt.Printf("%s  success\n", celebrate())
			return nil
		},
	}
}

//SuggestCmd suggests user share
func SuggestCmd() cli.Command {
	return cli.Command{
		Name:  "suggest",
		Usage: "Suggestes share between the group",
		Flags: suggestFlags(),
		// the action, or code that will be executed when
		// we execute our `ns` command
		Action: func(c *cli.Context) error {
			// a simple lookup function
			today := c.String("today")
			if today != "" {

				return nil
			}

			_, result := waitforinput("Apply blueprint? (yes/no)")
			if result {
				fmt.Printf("got the input")
			}

			time.Sleep(1 * time.Second)
			// we log the results to our console
			// using a trusty fmt.Println statement
			fmt.Printf("success")
			return nil
		},
	}
}

func waitforinput(title string) (string, bool) {
	yellow := color.New(color.FgYellow).SprintFunc()
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s  %s\n", devil(), yellow(title))
	for {
		fmt.Printf("%s   -> ", devil())
		text, _ := reader.ReadString('\n')
		// convert CRLF to LF
		text = strings.Replace(text, "\n", "", -1)
		fmt.Printf("%s  %s\n", devil(), "Loading ...")
		if strings.Compare("yes", text) == 0 {
			return text, true
		}
		return text, false
	}
}
