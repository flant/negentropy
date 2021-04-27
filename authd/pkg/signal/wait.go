package signal

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// WaitForProcessInterruption wait for SIGINT or SIGTERM and run a callback function.
//
// First signal start a callback function, which should call os.Exit(0).
// Next signal will call os.Exit(128 + signal-value).
func WaitForProcessInterruption(cb ...func()) {
	allowedCount := 1
	interruptCh := make(chan os.Signal, 1)

	forcedExit := func(s os.Signal) {
		fmt.Printf("Forced shutdown by '%s' signal\n", s.String())

		signum := 0
		switch v := s.(type) {
		case syscall.Signal:
			signum = int(v)
		}
		os.Exit(128 + signum)
	}

	signal.Notify(interruptCh, syscall.SIGINT, syscall.SIGTERM)
	for {
		sig := <-interruptCh
		allowedCount--
		switch allowedCount {
		case 0:
			if len(cb) > 0 {
				fmt.Printf("Grace shutdown by '%s' signal\n", sig.String())
				cb[0]()
			} else {
				forcedExit(sig)
			}
		case -1:
			forcedExit(sig)
		}
	}
}
