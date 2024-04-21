package miburi

import "encoding/json"

func exportJson(filename string) (string, error) {
	smiEntries, err := restoreObject(filename)
	if err != nil {
		return "", err
	}
	return exportJson_internal(smiEntries), nil
}

func exportJson_internal(smiEntries []SmiEntry) string {
	jsonBytes, _ := json.Marshal(smiEntries)
	return string(jsonBytes)
}
