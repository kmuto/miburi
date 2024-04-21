package miburi

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gosnmp/gosnmp"
)

func walk(opts *WalkCommand) error {
	smiEntries, err := makeSmiEntries(opts.Input)
	if err != nil {
		return err
	}

	gosnmp.Default.Target = opts.Host
	gosnmp.Default.Community = opts.Community
	gosnmp.Default.Port = opts.Port
	gosnmp.Default.Version = gosnmp.Version2c
	gosnmp.Default.Timeout = time.Duration(10 * time.Second)

	err = gosnmp.Default.Connect()
	if err != nil {
		return err
	}
	defer gosnmp.Default.Conn.Close()

	var walkedNodes []WalkedNode
	for _, oid := range opts.OIDs {
		walkedNodes = append(walkedNodes, exportObjectWalkedNode(smiEntries, oid, opts)...)
	}

	switch {
	case opts.Json:
		jsonBytes, _ := json.Marshal(walkedNodes)
		fmt.Println(string(jsonBytes))
	case opts.CSV:
		writer := csv.NewWriter(os.Stdout)
		defer writer.Flush()

		headers := []string{"OID", "Name", "MIB", "Type", "Value", "Enum", "Unit", "Desc"}
		err := writer.Write(headers)
		if err != nil {
			return err
		}
		for _, w := range walkedNodes {
			r := []string{w.OID, w.Name, w.MIB, w.Type, w.Value, w.Enum, w.Unit, w.Desc}
			err := writer.Write(r)
			if err != nil {
				return err
			}
		}
	default:
		printWalkedNodesAsText(walkedNodes)
	}

	return nil
}

func printWalkedNodesAsText(walkedNodes []WalkedNode) {
	for _, w := range walkedNodes {
		fmt.Printf("OID: %s\nName: %s\nMIB: %s\nType: %s\nValue: %s\n", w.OID, w.Name, w.MIB, w.Type, w.Value)

		if w.Enum != "" {
			fmt.Printf("Enum: %s\n", w.Enum)
		}
		if w.Unit != "" {
			fmt.Printf("Unit: %s\n", w.Unit)
		}
		if w.Desc != "" {
			fmt.Printf("Description: ---\n%s\n---\n", w.Desc)
		}

		fmt.Println()
	}
}

func exportObjectWalkedNode(smiEntries []SmiEntry, oid string, opts *WalkCommand) []WalkedNode {
	var walkedNodes []WalkedNode
	err := gosnmp.Default.BulkWalk(oid, func(pdu gosnmp.SnmpPDU) error {
		name, node := find(smiEntries, pdu.Name)
		walkedNode := WalkedNode{
			OID:  normalizeOid(pdu.Name),
			Name: name,
			MIB:  node.MIB,
		}

		switch pdu.Type {
		case gosnmp.OctetString:
			walkedNode.Type = "OctetString"
			v := pdu.Value.([]byte)
			if utf8.Valid(v) {
				walkedNode.Value = string(v)
			} else {
				s := "(hex) "
				for i, v := range v {
					if i > 0 {
						s = s + " "
					}
					s = s + fmt.Sprintf("%02x", v)
				}
				walkedNode.Value = s
			}
		case gosnmp.Integer:
			walkedNode.Type = "Integer"
			walkedNode.Value = fmt.Sprintf("%d", gosnmp.ToBigInt(pdu.Value).Int64())
		case gosnmp.ObjectIdentifier:
			walkedNode.Type = "ObjectIdentifier"
			walkedNode.Value = fmt.Sprintf("%s", pdu.Value)
		case gosnmp.IPAddress:
			walkedNode.Type = "IPAddress"
			walkedNode.Value = fmt.Sprintf("%s", pdu.Value)
		case gosnmp.Counter32:
			walkedNode.Type = "Counter32"
			walkedNode.Value = fmt.Sprintf("%d", gosnmp.ToBigInt(pdu.Value).Int64())
		case gosnmp.Gauge32:
			walkedNode.Type = "Gauge32"
			walkedNode.Value = fmt.Sprintf("%d", gosnmp.ToBigInt(pdu.Value).Int64())
		case gosnmp.TimeTicks:
			walkedNode.Type = "TimeTicks"
			walkedNode.Value = fmt.Sprintf("%d", gosnmp.ToBigInt(pdu.Value).Int64())
		case gosnmp.Counter64:
			walkedNode.Type = "Counter64"
			walkedNode.Value = fmt.Sprintf("%d", gosnmp.ToBigInt(pdu.Value).Int64())
		case gosnmp.Opaque:
			walkedNode.Type = "Opaque"
			walkedNode.Value = fmt.Sprintf("%s", pdu.Value)
		case gosnmp.NoSuchObject:
			walkedNode.Type = "NoSuchObject"
			walkedNode.Value = fmt.Sprintf("%s", pdu.Value)
		case gosnmp.NoSuchInstance:
			walkedNode.Type = "NoSuchInstance"
			walkedNode.Value = fmt.Sprintf("%s", pdu.Value)
		case gosnmp.EndOfMibView:
			return nil
		default:
			walkedNode.Type = "Unknown"
		}

		if opts.Verbose && name != "" {
			if node.SmiType != nil {
				if node.SmiType.Enum != nil {
					var enums []string
					for _, e := range node.SmiType.Enum.Values {
						enums = append(enums, fmt.Sprintf("%s = %v", e.Name, e.Value))
					}
					walkedNode.Enum = strings.Join(enums, ", ")
				}
				if node.SmiType.Units != "" {
					walkedNode.Unit = node.SmiType.Units
				}
			}
			walkedNode.Desc = node.Description
		}

		walkedNodes = append(walkedNodes, walkedNode)
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil
	}
	return walkedNodes
}
