package miburi

import (
	"fmt"
	"os"
	"regexp"

	"github.com/alexflint/go-arg"
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

type miburiOpts struct {
	DumpCommand *DumpCommand `arg:"subcommand:dump" help:"Dump MIB objects to file"`
	FindCommand *FindCommand `arg:"subcommand:find" help:"Find MIB object"`
	WalkCommand *WalkCommand `arg:"subcommand:walk" help:"Walk host"`
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
func (miburiOpts) Version() string {
	return fmt.Sprintf("miburi %s (rev: %s)", version, revision)
}

func Main() {
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
		fmt.Println("Dump completed")
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

func parseArgs(_ []string) (*miburiOpts, error) {
	var opts miburiOpts
	// XXX: MustParse uses args[1:] by default?
	arg.MustParse(&opts)
	if opts.DumpCommand == nil && opts.FindCommand == nil && opts.WalkCommand == nil && opts.JsonCommand == nil {
		return nil, fmt.Errorf("no command specified")
	}
	return &opts, nil
}

func normalizeOid(oid string) string {
	re := regexp.MustCompile(`^iso\.`)
	oid = re.ReplaceAllString(oid, ".1.")
	re = regexp.MustCompile(`^([^\.])`) // add dot to the beginning for forcing absolute OID
	oid = re.ReplaceAllString(oid, ".$1")
	return oid
}

func makeSmiEntries(filename string) ([]SmiEntry, error) {
	smiEntries, err := restoreObject(filename)
	if err != nil {
		return nil, err
	}
	return smiEntries, nil
}
