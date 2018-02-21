package core

import (
    "testing"
)

func TestNewCoreError(t *testing.T) {
	msg := "testing error"
	err := NewCoreError(ERR_ORPHAN_BLOCK, msg)
	if msg != err.Error() {
		t.Errorf("unexpected error message: Expecting '%s', Actual '%s'", msg, err.Error())
	}
	if ERR_ORPHAN_BLOCK != err.(*CoreError).Code() {
		t.Errorf("unexpected error code: Expecting '%d', Actual '%d'", ERR_ORPHAN_BLOCK, err.(*CoreError).Code())
	}
}

