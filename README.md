# Miburi

- [日本語](README-ja.md)

![](miburi.png)

## Description

**Miburi** is a tool for retrieving information from SNMP devices. It imports MIB files and displays information corresponding to the retrieved OID.

You can also export the information in JSON or CSV format, in addition to displaying it on the screen.

## Usage

```
Usage: miburi <command> [<args>]

Options:
  --help, -h             display this help and exit
  --version              display version and exit

Commands:
  dump                   Dump MIB objects to file
  find                   Find MIB object
  walk                   Walk host
  json                   Show MIB objects in JSON
```

### Dump MIB objects to file (please do this first)

```
Usage: miburi dump --directory DIRECTORY [--output OUTPUT]

Options:
  --directory DIRECTORY, -d DIRECTORY
                         Directory to scan for MIB files (multiple supported)
  --output OUTPUT, -o OUTPUT
                         Output file [default: smi_objects.gob]
  --help, -h             display this help and exit
  --version              display version and exit
```

Example:
```
$ miburi dump -d /usr/share/snmp/mibs -d /usr/share/snmp/mibs/iana -d /usr/share/snmp/mibs/ietf
  ...
Dump completed
```

Miburi parses each MIB file (MIB file should include `MIB` in its filename and be not compressed) from directories you specified and creates `smi_objects.gob` on current directory.

MIB files that fail to parse are ignored.

### Find MIB information using an OID

```
Usage: miburi find [--input INPUT] --target TARGET [--verbose]

Options:
  --input INPUT, -i INPUT
                         Input file [default: smi_objects.gob]
  --target TARGET, -t TARGET
                         OID to find (multiple supported)
  --verbose, -v          Verbose output
  --help, -h             display this help and exit
  --version              display version and exit
```

Example:
```
$ miburi find -t 1.3.6.1.2.1.1.4.0 -t iso.3.6.1.2.1.2.2.1.10.2 -v
OID: 1.3.6.1.2.1.1.4.0
Name: sysContact.0
MIB: SNMPv2-MIB
Type: DisplayString
Description: ---
The textual identification of the contact person for
this managed node, together with information on how
to contact this person.  If no contact information is
known, the value is the zero-length string.
---
OID: iso.3.6.1.2.1.2.2.1.10.2
Name: ifInOctets.2
MIB: RFC1213-MIB
Type: Counter32
Description: ---
The total number of octets received on the
interface, including framing characters.
---
```

Multiple OIDs can be specified with the `-t` option.

`-v` option displays Type, Unit, Enum, and Description (if present).

### Walk SNMP device

```
Usage: miburi walk [--input INPUT] [--host HOST] [--community COMMUNITY] [--port PORT] --target TARGET [--verbose] [--json] [--csv]

Options:
  --input INPUT, -i INPUT
                         Input file [default: smi_objects.gob]
  --host HOST, -H HOST   Host to walk [default: localhost]
  --community COMMUNITY, -c COMMUNITY
                         Community to walk [default: public]
  --port PORT, -p PORT   Port to walk [default: 161]
  --target TARGET, -t TARGET
                         OID to walk (multiple supported)
  --verbose, -v          Verbose output
  --json, -j             Output in JSON
  --csv, -C              Output in CSV
  --help, -h             display this help and exit
  --version              display version and exit
```

Runs snmpwalk on SNMP-enabled devices. Only SNMP v2c is supported.

The first `1.3.6` is at least required for the OID you specify.

```
$ miburi walk -H localhost -c public --target 1.3.6.1 -v
OID: 1.3.6.1.2.1.1.1.0
Name: sysDescr.0
MIB: SNMPv2-MIB
Type: OctetString
Value: Linux elemental 6.1.0-20-amd64 #1 SMP PREEMPT_DYNAMIC Debian 6.1.85-1 (20
24-04-11) x86_64

OID: 1.3.6.1.2.1.1.2.0
Name: sysObjectID.0
MIB: SNMPv2-MIB
Type: ObjectIdentifier
Value: .1.3.6.1.4.1.8072.3.2.10

OID: 1.3.6.1.2.1.1.3.0
Name: sysUpTimeInstance
MIB: DISMAN-EXPRESSION-MIB
Type: TimeTicks
Value: 1483518
  ...
```

An octetstring is converted to a string value; if it does not become a UTF-8 character, it is displayed in hexadecimal representation with (hex) appended.

You can also output the data in JSON or CSV.

```
$ miburi walk -H localhost -c public --target 1.3.6.1 -v -j | jq
[
  {
    "OID": "1.3.6.1.2.1.1.1.0",
    "Name": "sysDescr.0",
    "MIB": "SNMPv2-MIB",
    "Type": "OctetString",
    "Value": "Linux elemental 6.1.0-20-amd64 #1 SMP PREEMPT_DYNAMIC Debian 6.1.8
5-1 (2024-04-11) x86_64",
    "Enum": "",
    "Unit": "",
    "Desc": ""
  },
  {
    "OID": "1.3.6.1.2.1.1.2.0",
  ...
```

```
$ miburi walk -H localhost -c public --target 1.3.6.1 -v -C > localhost.csv
$ cat localhost.csv
OID,Name,MIB,Type,Value,Enum,Unit,Desc
1.3.6.1.2.1.1.1.0,sysDescr.0,SNMPv2-MIB,OctetString,Linux ...
64 #1 SMP PREEMPT_DYNAMIC Debian 6...
1.3.6.1.2.1.1.2.0,sysObjectID.0,SNMPv2-MIB,ObjectIdentifier,.1.3.6.1.4.1.8072.3.2.10,,,
1.3.6.1.2.1.1.3.0,sysUpTimeInstance,DISMAN-EXPRESSION-MIB,TimeTicks,1504062,,,
  ...
```

### Show MIB information in JSON

```
Usage: miburi json [--input INPUT]

Options:
  --input INPUT, -i INPUT
                         Input file [default: smi_objects.gob]
  --help, -h             display this help and exit
  --version              display version and exit
```

Stored MIB information can be output in JSON format.

```
$ miburi json | jq
[
  {
    "Module": {
      "ContactInfo": "Primary Contact: M J Oldfield\nemail:     m@mail.tc",
      "Description": "This MIB module defines objects for lm_sensor derived data
.",
      "Language": "SMIv2",
      "Name": "LM-SENSORS-MIB",
      "Organization": "AdamsNames Ltd",
      "Path": "/usr/share/snmp/mibs/LM-SENSORS-MIB.txt",
      "Reference": ""
    },
    "Nodes": [
      {
        "Access": "Unknown",
        "Decl": "ValueAssignment",
        "Description": "",
        "Kind": "Node",
        "Name": "lmSensors",
  ...
```

## Retrieving MIB files

You can use Docker to retrieve the major MIB files.

```
$ docker image build -t kmuto/mib:latest .
$ docker container run --rm -v ".:/work" kmuto/mib
```

`mibs` directory will be created and have MIB files. Then do:

```
$ miburi dump -d mibs
```

## License

© 2024 Kenshi Muto

MIT License (see [LICENSE](LICENSE) file)
