package cmd

import (
	"fmt"
	"strconv"
	"strings"
)

func devil() string {
	s, err := unquoteCodePoint("\\U0001f47f")
	if err != nil {
		fmt.Println("Problem while printing devil", err)
	}
	return s
}

func unquoteCodePoint(s string) (string, error) {
	r, err := strconv.ParseInt(strings.TrimPrefix(s, "\\U"), 16, 32)
	return string(r), err
}

func validating() string {
	s, err := unquoteCodePoint("\\U000231B")
	if err != nil {
		fmt.Println("Problem while printing validating", err)
	}
	return s
}

func celebrate() string {
	s, err := unquoteCodePoint("\\U0001f38A")
	if err != nil {
		fmt.Println("Problem while printing celebrate", err)
	}
	return s
}
