package radiusconfig

// buildMSCHAP emits a minimal mschap module configuration (no NTDOM domain hack).
func buildMSCHAP() string {
	return `mschap {
	use_mppe = yes
	require_encryption = yes
	require_strong = yes
}
`
}
