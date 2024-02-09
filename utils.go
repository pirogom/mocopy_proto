package main

import (
	"os"
	"time"

	"github.com/sqweek/dialog"
)

/**
*	timeGetTime
**/
func timeGetTime() int64 {
	return time.Now().UnixNano() / 1000000
}

/**
*	openPathDlg
**/
func openPathDlg(title string) (string, error) {
	result, err := dialog.Directory().Title(title).Browse()

	if err != nil {
		return "", err
	}
	return result, nil
}

/**
*	isExistFile
**/
func isExistFile(fname string) bool {
	if _, err := os.Stat(fname); os.IsNotExist(err) {
		return false
	}
	return true
}
