package service

func addItemToUniqueArray(uniqueHelperMap map[string]bool, uniqueArrayOfStrings *[]string, s string) {
	if uniqueHelperMap[s] {
		return
	}
	*uniqueArrayOfStrings = append(*uniqueArrayOfStrings, s)
	uniqueHelperMap[s] = true
}
