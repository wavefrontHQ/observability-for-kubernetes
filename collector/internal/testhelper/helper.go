package testhelper

// Convenience function for taking the address of a uint64 literal.
func Uint64Val(seed, offset int) *uint64 {
	val := uint64(seed + offset)
	return &val
}
