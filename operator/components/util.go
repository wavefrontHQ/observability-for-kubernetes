package components

import (
	"crypto/sha1"
	"fmt"
)

func HashValue(bytes []byte) string {
	h := sha1.New()
	h.Write(bytes)

	return fmt.Sprintf("%x", h.Sum(nil))
}
