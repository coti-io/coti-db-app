package service

func addItemToUniqueArray(uniqueHelperMap map[string]bool, uniqueArrayOfStrings *[]string, s string) {
	if uniqueHelperMap[s] {
		return
	}
	*uniqueArrayOfStrings = append(*uniqueArrayOfStrings, s)
	uniqueHelperMap[s] = true
}

func increaseCountIfUnique(uniqueHelperMap map[string]bool, stringCounter map[string]int32, keyToCheck string, keyToIncrease string) {
	if uniqueHelperMap[keyToCheck] {
		return
	}
	stringCounter[keyToIncrease] = stringCounter[keyToIncrease] + 1
	uniqueHelperMap[keyToCheck] = true
}
