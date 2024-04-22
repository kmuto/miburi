package miburi

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"

	"github.com/sleepinggenius2/gosmi"
)

const DBVersion = 1

type DBObject struct {
	Version    int
	SmiEntries []SmiEntry
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

func dumpObject(fileName string, smiEntries []SmiEntry) error {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	err := enc.Encode(&DBObject{
		Version:    DBVersion,
		SmiEntries: smiEntries})
	if err != nil {
		return err
	}

	err = os.WriteFile(fileName, b.Bytes(), 0644)
	if err != nil {
		return err
	}

	return nil
}

func restoreObject(filename string) ([]SmiEntry, error) {
	var dbObject DBObject
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	dec := gob.NewDecoder(bytes.NewReader(data))
	err = dec.Decode(&dbObject)
	if err != nil {
		return nil, err
	}
	if dbObject.Version != DBVersion {
		return nil, fmt.Errorf("%s is incompatible with this miburi. Please recreate it (see `miburi dump --help`)", filename)
	}

	return dbObject.SmiEntries, nil
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
}

func getMIBNames(paths []string) ([]string, error) {
	var mibs []string
	for _, path := range paths {
		files, err := os.ReadDir(path)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			if !file.IsDir() {
				mibs = append(mibs, file.Name())
			}
		}
	}
	return mibs, nil
}
