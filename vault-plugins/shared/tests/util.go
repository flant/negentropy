package tests

import (
	"fmt"
	"time"
)

func Repeat(f func() error, maxAttempts int) error {
	err := f()
	counter := 1
	for err != nil {
		if counter > maxAttempts {
			return fmt.Errorf("exceeded attempts, last err:%w", err)
		}
		fmt.Printf("waiting fail %d attempt\n", counter)
		time.Sleep(time.Second)
		counter++
		err = f()
	}
	fmt.Printf("waiting completed successfully, attempt %d\n", counter)
	return nil
}
