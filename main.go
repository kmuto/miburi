package main

import (
	"bytes"
	"encoding/csv"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/alexflint/go-arg"
	"github.com/gosnmp/gosnmp"
	"github.com/sleepinggenius2/gosmi"
)

type SmiNodeWithIndex struct {
	gosmi.SmiNode
	OIDString string
	MIB       string
}

type SmiEntry struct {
	Module gosmi.SmiModule
	Nodes  []SmiNodeWithIndex
	Types  []gosmi.SmiType
}

type DumpCommand struct {
	Directories []string `arg:"-d,--directory,separate,required" help:"Directory to scan for MIB files (multiple supported)"`
	Output      string   `arg:"-o,--output" help:"Output file" default:"smi_objects.gob"`
}

type FindCommand struct {
	Input   string   `arg:"-i,--input" help:"Input file" default:"smi_objects.gob"`
	OIDs    []string `arg:"-t,--target,separate,required" help:"OID to find (multiple supported)"`
	Verbose bool     `arg:"-v,--verbose" help:"Verbose output"`
}

type JsonCommand struct {
	Input string `arg:"-i,--input" help:"Input file" default:"smi_objects.gob"`
}

type WalkCommand struct {
	Input     string   `arg:"-i,--input" help:"Input file" default:"smi_objects.gob"`
	Host      string   `arg:"-H,--host" help:"Host to walk" default:"localhost"`
	Community string   `arg:"-c,--community" help:"Community to walk" default:"public"`
	Port      uint16   `arg:"-p,--port" help:"Port to walk" default:"161"`
	OIDs      []string `arg:"-t,--target,separate,required" help:"OID to walk (multiple supported)"`
	Verbose   bool     `arg:"-v,--verbose" help:"Verbose output"`
	Json      bool     `arg:"-j,--json" help:"Output in JSON"`
	CSV       bool     `arg:"-C,--csv" help:"Output in CSV"`
}

type ssagasuOpts struct {
	DumpCommand *DumpCommand `arg:"subcommand:dump" help:"Dump MIB objects to file"`
	FindCommand *FindCommand `arg:"subcommand:find" help:"Find MIB object by OID"`
	WalkCommand *WalkCommand `arg:"subcommand:walk" help:"Walk host by OID"`
	JsonCommand *JsonCommand `arg:"subcommand:json" help:"Show MIB objects in JSON"`
}

type WalkedNode struct {
	OID   string
	Name  string
	MIB   string
	Type  string
	Value string
	Enum  string
	Unit  string
	Desc  string
}

var version string
var revision string

// interface implementation for go-arg
func (ssagasuOpts) Version() string {
	return fmt.Sprintf("ssagasu %s (rev: %s)", version, revision)
}

func main() {
	opts, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	gosmi.Init()
	defer gosmi.Exit()

	switch {
	case opts.DumpCommand != nil:
		err := createDump(opts.DumpCommand.Output, opts.DumpCommand.Directories)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(`Dump completed`)
	case opts.FindCommand != nil:
		smiEntries, err := makeSmiEntries(opts.FindCommand.Input)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		for _, oid := range opts.FindCommand.OIDs {
			exportTextFindedNode(smiEntries, oid, opts.FindCommand)
		}
	case opts.WalkCommand != nil:
		err := walk(opts.WalkCommand)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(0)
		}
	case opts.JsonCommand != nil:
		json, err := exportJson(opts.JsonCommand.Input)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(json)
	}
}

func parseArgs(_ []string) (*ssagasuOpts, error) {
	var opts ssagasuOpts
	// XXX: MustParse uses args[1:] by default?
	arg.MustParse(&opts)
	if opts.DumpCommand == nil && opts.FindCommand == nil && opts.WalkCommand == nil && opts.JsonCommand == nil {
		return nil, fmt.Errorf("no command specified")
	}
	return &opts, nil
}

func createDump(filename string, paths []string) error {
	smiEntries, err := loadModules(paths)
	if err != nil {
		return err
	}
	err = dumpObject(filename, smiEntries)
	if err != nil {
		return err
	}
	return nil
}

func makeSmiEntries(filename string) ([]SmiEntry, error) {
	smiEntries, err := restoreObject(filename)
	if err != nil {
		return nil, err
	}
	return smiEntries, nil
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

func normalizeOid(oid string) string {
	re := regexp.MustCompile(`^\.`)
	oid = re.ReplaceAllString(oid, "")
	re = regexp.MustCompile(`^iso\.`)
	oid = re.ReplaceAllString(oid, "1.")
	return oid
}

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

func loadModules(paths []string) ([]SmiEntry, error) {
	var modules []string
	var smiEntries []SmiEntry

	modules, err := getMIBNames(paths)
	if err != nil {
		return nil, err
	}

	for _, p := range paths {
		gosmi.AppendPath(p)
	}

	for i, m := range modules {
		moduleName, err := gosmi.LoadModule(m)
		if err != nil {
			fmt.Printf("Load error (skip): %s\n", err)
			continue
		}
		modules[i] = moduleName
	}

	for _, module := range modules {
		m, err := gosmi.GetModule(module)
		if err != nil {
			fmt.Printf("ModuleTrees Error: %s\n", err)
			continue
		}

		sminodes := m.GetNodes()
		var nodes []SmiNodeWithIndex
		for _, node := range sminodes {
			nodes = append(nodes, SmiNodeWithIndex{
				SmiNode:   node,
				OIDString: node.Oid.String(),
				MIB:       m.Name,
			})
		}
		types := m.GetTypes()

		smiEntry := SmiEntry{
			Module: m,
			Nodes:  nodes,
			Types:  types,
		}

		smiEntries = append(smiEntries, smiEntry)
	}
	return smiEntries, nil
	//jsonBytes, _ := json.Marshal(snmpEntry)
	//os.Stdout.Write(jsonBytes)
}

func getMIBNames(paths []string) ([]string, error) {
	var mibs []string
	for _, path := range paths {
		files, err := os.ReadDir(path)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			if !file.IsDir() && strings.Contains(file.Name(), "MIB") {
				mibs = append(mibs, file.Name())
			}
		}
	}
	return mibs, nil
}

func dumpObject(fileName string, snmpEntries []SmiEntry) error {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	err := enc.Encode(snmpEntries)
	if err != nil {
		return err
	}

	// save to file
	err = os.WriteFile(fileName, b.Bytes(), 0644)
	if err != nil {
		return err
	}

	return nil
}

func restoreObject(filename string) ([]SmiEntry, error) {
	var smiEntries []SmiEntry
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	dec := gob.NewDecoder(bytes.NewReader(data))
	err = dec.Decode(&smiEntries)
	if err != nil {
		return nil, err
	}

	return smiEntries, nil
}
