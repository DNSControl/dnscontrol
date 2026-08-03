package models

import (
	"math"

	dnsv2 "codeberg.org/miekg/dns"
)

// ReverseLatitude takes the packed latitude and returns the hemisphere, degrees, minutes, and seconds.
func ReverseLatitude(lat uint32) (string, uint8, uint8, float64) {
	var hemisphere string
	if lat >= dnsv2.LOCEquator {
		hemisphere = "N"
		lat = lat - dnsv2.LOCEquator
	} else {
		hemisphere = "S"
		lat = dnsv2.LOCEquator - lat
	}
	degrees := uint8(lat / dnsv2.LOCDegrees)
	lat -= uint32(degrees) * dnsv2.LOCDegrees
	minutes := uint8(lat / dnsv2.LOCHours)
	lat -= uint32(minutes) * dnsv2.LOCHours
	seconds := float64(lat) / 1000

	return hemisphere, degrees, minutes, seconds
}

// ReverseLongitude takes the packed longitude and returns the hemisphere, degrees, minutes, and seconds.
func ReverseLongitude(lon uint32) (string, uint8, uint8, float64) {
	var hemisphere string
	if lon >= dnsv2.LOCPrimemeridian {
		hemisphere = "E"
		lon = lon - dnsv2.LOCPrimemeridian
	} else {
		hemisphere = "W"
		lon = dnsv2.LOCPrimemeridian - lon
	}
	degrees := uint8(lon / dnsv2.LOCDegrees)
	lon -= uint32(degrees) * dnsv2.LOCDegrees
	minutes := uint8(lon / dnsv2.LOCHours)
	lon -= uint32(minutes) * dnsv2.LOCHours
	seconds := float64(lon) / 1000

	return hemisphere, degrees, minutes, seconds
}

// ReverseAltitude takes the packed altitude and returns the altitude in meters.
func ReverseAltitude(packedAltitude uint32) float64 {
	return float64(packedAltitude)/100 - 100000
}

// ReverseENotationInt produces a number from a mantissa_exponent 4bits:4bits uint8.
func ReverseENotationInt(packedValue uint8) float64 {
	mantissa := float64((packedValue >> 4) & 0x0F)
	exponent := int(packedValue & 0x0F)

	centimeters := mantissa * math.Pow10(exponent)

	// Return in meters
	return centimeters / 100
}
