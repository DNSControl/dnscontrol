package js

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"

	"github.com/robertkrimen/otto"
)

// Exposes sha1, sha256, and sha512 hashing functions to Javascript.
func hashFunc(call otto.FunctionCall) otto.Value {
	if len(call.ArgumentList) != 2 {
		throw(call.Otto, "require takes exactly two arguments")
	}
	algorithm := call.Argument(0).String() // The algorithm to use for hashing
	value := call.Argument(1).String()     // The value to hash
	result := otto.Value{}
	// fmt.Printf("%s\n", value)

	switch algorithm {
	case "SHA1", "sha1":
		tmp := sha1.New()
		tmp.Write([]byte(value))
		// fmt.Printf("%s\n", hex.EncodeToString(tmp.Sum(nil)))
		result, _ = otto.ToValue(hex.EncodeToString(tmp.Sum(nil)))
	case "SHA256", "sha256":
		tmp := sha256.New()
		tmp.Write([]byte(value))
		result, _ = otto.ToValue(hex.EncodeToString(tmp.Sum(nil)))
	case "SHA512", "sha512":
		tmp := sha512.New()
		tmp.Write([]byte(value))
		result, _ = otto.ToValue(hex.EncodeToString(tmp.Sum(nil)))
	default:
		throw(call.Otto, fmt.Sprintf("invalid algorithm %s given", algorithm))
	}
	return result
}
