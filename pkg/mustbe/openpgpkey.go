package mustbe

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// OpenPGPKey returns the key re-encoded as base64. The input may be hex or base64.
func OpenPGPKey(a any) (string, error) {
	switch v := a.(type) {
	case string:
		return openPGPKeyFromString(v)
	}
	return "", fmt.Errorf("not a valid OPENPGPKEY: %v", a)
}

// openPGPKeyFromString decodes an OPENPGP key and re-encodes it as base64.
func openPGPKeyFromString(encodedKey string) (string, error) {
	// Decode the key, which can be either hex or base64.
	decodedKey, err := decodeHexOrBase64(encodedKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode OPENPGPKEY: %w", err)
	}

	// Re-encode the key as base64, since the input may have been hex.
	encodedKey = b64.EncodeToString(decodedKey)
	return encodedKey, nil
}

var b64 = base64.StdEncoding.Strict()

// The target of an OPENPGPKEY record can be either hex or base64, so we need to
// be able to decode both formats.
//
// PGP keys are quite long and are largely random, so the odds of the base64
// encoding of a PGP key also being valid hex are very low. Therefore, we will
// assume that if a string is both valid hex and valid base64, then it is hex.
// The other cases where a string decodes as only hex, only base64, or neither
// are all unambiguous.
func decodeHexOrBase64(s string) ([]byte, error) {
	// A string with mixed casing *could* be hex, but it's more likely to be
	// base64. So we only try to decode as hex only if the string is all a
	// single case.
	var (
		hexErr  error
		hexData []byte
	)
	if s == strings.ToLower(s) || s == strings.ToUpper(s) {
		hexData, hexErr = hex.DecodeString(s)
	} else {
		hexData, hexErr = nil, fmt.Errorf(
			"hex string contains mixed case: %#v", s,
		)
	}

	// Also try to decode the string as base64, using the strictest possible
	// decoder (which is also used in the DNS "presentation" format and the "gpg
	// --armor" format).
	var (
		b64Err  error
		b64Data []byte
	)
	// Reject base64 strings that contain whitespace
	if strings.ContainsAny(s, " \t\r\n") {
		b64Data, b64Err = nil, fmt.Errorf(
			"base64 string contains whitespace: %#v", s,
		)
	} else {
		b64Data, b64Err = b64.DecodeString(s)
	}

	// Return the result.
	if hexErr != nil && b64Err != nil {
		// Both decodings failed, so there's nothing that we can do but return
		// an error.
		return nil, fmt.Errorf(
			"string is neither valid hex nor valid base64: %w; %w",
			hexErr, b64Err,
		)
	} else if hexErr == nil && b64Err != nil {
		// Only the hex decoding succeeded, therefore the input must only be
		// valid hex.
		return hexData, nil
	} else if hexErr != nil && b64Err == nil {
		// Only the base64 decoding succeeded, therefore the input must only be
		// valid base64.
		return b64Data, nil
	} else if hexErr == nil && b64Err == nil {
		// Both decodings succeeded. This is theoretically ambiguous, but it's
		// very unlikely that a valid base64 string would also be valid hex, so
		// we will assume that the input is hex.
		return hexData, nil
	} else {
		return nil, errors.New("unreachable")
	}
}
