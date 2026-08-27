package mustbe

import "testing"

// Uint32

func TestUint32FromUint(t *testing.T) {
	if got := Uint32(uint(8888)); got != 8888 {
		t.Fatalf("Uint32(uint(8888)) = %d, want 8888", got)
	}
}

func TestUint32FromInt(t *testing.T) {
	if got := Uint32(int(7777)); got != 7777 {
		t.Fatalf("Uint32(int(7777)) = %d, want 7777", got)
	}
}

func TestUint32RejectsNegative(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected function to panic")
		}
	}()

	Uint32(int(-1))
}

// Int64

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
