package mustbe

import (
	"fmt"
	"math"
	"strconv"
)

func Uint8(arg any) uint8 {
	switch v := arg.(type) {
	case uint8:
		return v
	case uint16:
		if v > math.MaxUint8 {
			panic(fmt.Sprintf("value %v overflows uint8", arg))
		}
		return uint8(v)
	case int16:
		if v < 0 || v > math.MaxUint8 {
			panic(fmt.Sprintf("value %v overflows uint8", arg))
		}
		return uint8(v)
	case uint32:
		if v > math.MaxUint8 {
			panic(fmt.Sprintf("value %v overflows uint8", arg))
		}
		return uint8(v)
	case uint:
		if v > math.MaxUint8 {
			panic(fmt.Sprintf("value %v overflows uint8", arg))
		}
		return uint8(v)
	case int:
		if v < 0 || v > math.MaxUint8 {
			panic(fmt.Sprintf("value %v overflows uint8", arg))
		}
		return uint8(v)
	case float64:
		if v < 0 || v > math.MaxUint8 {
			panic(fmt.Sprintf("value %v overflows uint8", arg))
		}
		return uint8(v)
	case string:
		ni, err := strconv.ParseUint(arg.(string), 10, 8)
		if err != nil {
			panic(fmt.Sprintf("value %v is not a number (uint8 wanted)", arg))
		}
		return uint8(ni)
	}
	panic(fmt.Sprintf("value %v is type %T, expected uint8", arg, arg))
}

func Uint16(arg any) uint16 {
	switch v := arg.(type) {
	case uint8:
		return uint16(v)
	case uint16:
		return v
	case int16:
		if v < 0 {
			panic(fmt.Sprintf("value %v underflows uint16", arg))
		}
	case uint:
		if v > math.MaxUint16 {
			panic(fmt.Sprintf("value %v overflows uint16", arg))
		}
		return uint16(v)
	case uint32:
		if v > math.MaxUint16 {
			panic(fmt.Sprintf("value %v overflows uint16", arg))
		}
		return uint16(v)
	case uint64:
		if v > math.MaxUint16 {
			panic(fmt.Sprintf("value %v overflows uint16", arg))
		}
		return uint16(v)
	case int:
		if v < 0 || v > math.MaxUint16 {
			panic(fmt.Sprintf("value %v overflows uint16", arg))
		}
		return uint16(v)
	case int64:
		if v < 0 || v > math.MaxUint16 {
			panic(fmt.Sprintf("value %v overflows uint16", arg))
		}
		return uint16(v)
	case float64:
		if v < 0 || v > math.MaxUint16 {
			panic(fmt.Sprintf("value %v overflows uint16", arg))
		}
		return uint16(v)
	case string:
		ni, err := strconv.ParseUint(arg.(string), 10, 16)
		if err != nil {
			panic(fmt.Sprintf("value %v is not a number (uint16 wanted)", arg))
		}
		return uint16(ni)
	}
	panic(fmt.Sprintf("value %v is type %T, expected uint16", arg, arg))
}

func Uint32(arg any) uint32 {
	switch v := arg.(type) {
	case uint8:
		return uint32(v)
	case uint16:
		return uint32(v)
	case int16:
		if v < 0 {
			panic(fmt.Sprintf("value %v underflows uint32", arg))
		}
		return uint32(v)
	case uint32:
		return v
	case uint:
		if v > math.MaxUint32 {
			panic(fmt.Sprintf("value %v overflows uint32", arg))
		}
		return uint32(v)
	case int:
		if v < 0 || v > math.MaxUint32 {
			panic(fmt.Sprintf("value %v overflows uint32", arg))
		}
		return uint32(v)
	case float64:
		if v < 0 || v > math.MaxUint32 {
			panic(fmt.Sprintf("value %v overflows uint32", arg))
		}
		return uint32(v)
	case string:
		ni, err := strconv.ParseUint(arg.(string), 10, 32)
		if err != nil {
			panic(fmt.Sprintf("value %v is not a number (uint32 wanted)", arg))
		}
		return uint32(ni)
	}
	panic(fmt.Sprintf("value %v is type %T, expected uint32", arg, arg))
}

func Uint64(arg any) uint64 {
	switch v := arg.(type) {
	case string:
		num, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			panic(fmt.Sprintf("value %v is not a number (uint64 wanted)", v))
		}
		return uint64(num)
	case uint64:
		return uint64(v)
	}
	panic(fmt.Sprintf("value %v is type %T, expected uint64", arg, arg))
}

func Int64(arg any) int64 {
	switch v := arg.(type) {
	case int64:
		return v
	case uint16:
		return int64(v)
	case float64:
		if math.Trunc(v) != v || v < math.MinInt64 || v >= float64(math.MaxInt64) {
			panic(fmt.Sprintf("value %v is not an int64", arg))
		}
		return int64(v)
	case string:
		ni, err := strconv.ParseInt(arg.(string), 10, 64)
		if err != nil {
			panic(fmt.Sprintf("value %v is not a number (int64 wanted)", arg))
		}
		return int64(ni)
	}
	panic(fmt.Sprintf("value %v is type %T, expected int64", arg, arg))
}

func Float32(arg any) float32 {
	switch v := arg.(type) {
	case float32:
		return v
	case string:
		f, err := strconv.ParseFloat(v, 32)
		if err != nil {
			panic(fmt.Sprintf("string to float parse error: %q err=%s", v, err))
		}
		if f < -math.MaxFloat32 || f > math.MaxFloat32 {
			panic(fmt.Sprintf("value %q overflows float32", arg))
		}
		return float32(f)
	case float64:
		if v < -math.MaxFloat32 || v > math.MaxFloat32 {
			panic(fmt.Sprintf("value %v overflows float32", arg))
		}
		return float32(v)
	}
	panic(fmt.Sprintf("value %v is type %T, expected float32", arg, arg))
}

func Float64(arg any) float64 {
	switch v := arg.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	}
	panic(fmt.Sprintf("value %v is type %T, expected float64", arg, arg))
}
