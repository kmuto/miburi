package miburi

import (
	"fmt"
	"os"
	"strings"
)

func find(smiEntries []SmiEntry, oid string) (string, SmiNodeWithIndex) {
	oidMap := makeOidMap(smiEntries)
	oidname, node := findNodeByOID(oidMap, normalizeOid(oid))
	return oidname, node
}

func findNodeByOID(oidMap map[string]SmiNodeWithIndex, oid string) (string, SmiNodeWithIndex) {
	s := oid
	tail := ""
	for {
		// Well, ugly but works
		if oidMap[s].OIDString == "" {
			i := strings.LastIndex(s, ".")
			if i < 0 {
				return "", SmiNodeWithIndex{}
			}
			tail = s[i:] + tail
			s = s[:i]
			if s == "" {
				return "", SmiNodeWithIndex{}
			}
		} else {
			return (oidMap[s].Name + tail), oidMap[s]
		}
	}
}

func makeOidMap(smiEntries []SmiEntry) map[string]SmiNodeWithIndex {
	oidMap := make(map[string]SmiNodeWithIndex)
	for _, smiEntry := range smiEntries {
		for _, node := range smiEntry.Nodes {
			oidMap[node.OIDString] = node
		}
	}
	return oidMap
}

func exportTextFindedNode(smiEntries []SmiEntry, oid string, opts *FindCommand) {
	name, node := find(smiEntries, oid)
	if name == "" {
		fmt.Fprintf(os.Stderr, "name not found for OID: %s\n", oid)
		return
	}
	fmt.Printf("OID: %s\nName: %s\nMIB: %s\n", oid, name, node.MIB)
	if opts.Verbose {
		fmt.Printf("Type: %s\n", node.SmiType.Name)
		if node.SmiType.Enum != nil {
			var enums []string
			for _, e := range node.SmiType.Enum.Values {
				enums = append(enums, fmt.Sprintf("%s = %v", e.Name, e.Value))
			}
			fmt.Printf("Enum: %s\n", strings.Join(enums, ", "))
		}
		if node.SmiType.Units != "" {
			fmt.Printf("Unit: %s\n", node.SmiType.Units)
		}
		fmt.Printf("Description: ---\n%s\n---\n", node.Description)
	}
}
