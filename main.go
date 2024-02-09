package main

import (
	"runtime"

	"github.com/pirogom/walkmgr"
)

const (
	MOCOPY_VER = "0.01"
)

/**
*	main
**/
func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	walkmgr.SetUseWalkPositionMgr()
	walkmgr.LoadIcon(embedMoCopyIcon, embedMoCopyIconName)

	NewMainWin()
}
