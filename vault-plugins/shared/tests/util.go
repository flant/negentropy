package tests

import (
	"fmt"
	"runtime"
	"time"
)

// SlowRepeat for checks in case vaults should restore after down
func SlowRepeat(f func() error, maxAttempts int) error {
	return customizedRepeat(f, maxAttempts, time.Second*10)
}

// Repeat for checks in case data should go through kafka etc
func Repeat(f func() error, maxAttempts int) error {
	return customizedRepeat(f, maxAttempts, time.Second)
}

// FastRepeat for quick checks, for using in multy goroutines checks
func FastRepeat(f func() error, maxAttempts int) error {
	return customizedRepeat(f, maxAttempts, time.Millisecond*50)
}

func customizedRepeat(f func() error, maxAttempts int, duration time.Duration) error {
	err := f()
	counter := 1
	for err != nil {
		runtime.Gosched()
		if counter > maxAttempts {
			return fmt.Errorf("exceeded attempts, last err:%w", err)
		}
		fmt.Printf("waiting fail %d attempt\n", counter)
		time.Sleep(duration)
		counter++
		err = f()
	}
	fmt.Printf("waiting completed successfully, attempt %d\n", counter)
	return nil
}
