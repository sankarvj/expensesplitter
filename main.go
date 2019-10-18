package main

import (
	"log"
	"os"

	"github.com/sankarvj/expensesplitter/cmd"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "Split your expense"
	app.Usage = "Command line tool to split expense among your group. You can also send emails and remainders using this tool"
	app.Commands = commands()

	// start our application
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func commands() []cli.Command {
	return []cli.Command{
		cmd.TransactionCmd(),
		cmd.SuggestCmd(),
	}
}
