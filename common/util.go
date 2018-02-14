package common

import (
	"time"
)

func RunTimeBound(sec time.Duration, method func () error, timeoutError error) error {
	var err error
	// create a channel to signal done
	done := make(chan struct{})
	// start a timer
	wait := time.NewTimer(sec * time.Second)
	defer wait.Stop()
	// invoke the method in a go routine
	go func(){
		err = method() 
		done <- struct{}{}
	}()
	// wait for either done, or timeout
	select {
		case <- done:
			break
		case <- wait.C:
			err = timeoutError
	}
	return err
}