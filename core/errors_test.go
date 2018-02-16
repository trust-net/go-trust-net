package core

import (
    "testing"
)

func TestNewCoreError(t *testing.T) {
	msg := "testing error"
	err := NewCoreError(msg)
	if msg != err.Error() {
		t.Errorf("unexpected error message: Expecting '%s', Actual '%s'", msg, err.Error())
	}
}

