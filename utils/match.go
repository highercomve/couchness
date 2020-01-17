package utils

// Divmod get divisisor and module
func Divmod(numerator, denominator int) (quotient, remainder int) {
	quotient = numerator / denominator // integer division, decimals are truncated
	remainder = numerator % denominator
	return quotient, remainder
}

// DivCeil get next to quotient
func DivCeil(numerator, denominator int) (quotient int) {
	quotient = numerator / denominator // integer division, decimals are truncated
	remainder := numerator % denominator
	if remainder > 0 {
		quotient++
	}
	return quotient
}
