package common

import (
    "testing"
    "time"
)

type testError struct {}

func (e *testError) Error() string {
	return "test error"
}


func TestRunTimeBoundOnTimeout(t *testing.T) {
	// invoke a method that takes longer than timeout delay
	timeout, wait := time.Duration(1), time.Duration(5)
	if err := RunTimeBound(timeout, func() error {time.Sleep(wait*time.Second); return nil}, &testError{}); err == nil {
		t.Errorf("timed out unexpectedly")
	}
}


func TestRunTimeBoundWithinTime(t *testing.T) {
	// invoke a method that takes shorter than timeout delay
	timeout, wait := time.Duration(5), time.Duration(1)
	if err := RunTimeBound(timeout, func() error {time.Sleep(wait*time.Second); return nil}, &testError{}); err != nil {
		t.Errorf("did not timeout")
	}
}
