package handlers

// truncateForLog returns a short, length-safe prefix of s for logging. Avoids
// slice-bounds panics on short inputs (e.g. malformed OAuth params).
func truncateForLog(s string) string {
	const n = 10
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
