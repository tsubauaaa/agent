package main

import (
	"fmt"

	"github.com/tsubauaaa/agent/cmd"
)

func main() {
	errs := make(chan error, 5)
	exitCh := make(chan struct{})

	go func() {
		for {
			err := <-errs
			if err != nil {
				fmt.Print(err)
			}
		}
	}()

	// Agentサービスのメインループ処理の開始
	cmd.MainLoop(errs, exitCh)
}
