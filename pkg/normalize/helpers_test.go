package normalize

import (
	"testing"
)

func checkError(t *testing.T, err error, shouldError bool, experiment string) {
	if err != nil && !shouldError {
		t.Errorf("%v: Error (%v)\n", experiment, err)
	}
	if err == nil && shouldError {
		t.Errorf("%v: Expected error but got none \n", experiment)
	}
}
