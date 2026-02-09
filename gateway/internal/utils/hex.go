package utils

// Hex4 formats a uint16 as a 4-character hexadecimal string (e.g., "FFFF")
// Helper for 0xFFFF formatting without pulling in fmt everywhere in logs
func Hex4(v uint16) string {
	const hexd = "0123456789ABCDEF"
	return string([]byte{
		hexd[(v>>12)&0xF],
		hexd[(v>>8)&0xF],
		hexd[(v>>4)&0xF],
		hexd[v&0xF],
	})
}

// BytesToHex converts a byte slice to a hexadecimal string
func BytesToHex(b []byte) string {
	const hexd = "0123456789ABCDEF"
	out := make([]byte, 0, len(b)*2)
	for _, x := range b {
		out = append(out, hexd[x>>4], hexd[x&0x0F])
	}
	return string(out)
}
