package mustbe

import "testing"

func TestInt64FromFloat64(t *testing.T) {
	if got := Int64(float64(12345)); got != 12345 {
		t.Fatalf("Int64(12345) = %d, want 12345", got)
	}
}

func TestInt64RejectsFractionalFloat64(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Int64(123.5) did not panic")
		}
	}()
	Int64(123.5)
}
