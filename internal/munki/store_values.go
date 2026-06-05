package munki

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
