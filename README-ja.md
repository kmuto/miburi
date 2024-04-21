# Miburi

- [English](README.md)

## 概要

**Miburi**は、SNMP対応デバイスから情報を取得するためのツールです。MIBファイルをもとに、取得したOIDに対応付けられた情報を表示します。

通常の画面表示のほか、JSONまたはCSV形式で情報を出力することもできます。

## 使い方

```
使い方: miburi <コマンド> [<引数>]

オプション:
  --help, -h             ヘルプを表示して終了
  --version              バージョンを表示して終了

コマンド:
  dump                   MIBオブジェクトをファイルにダンプする
  find                   MIBオブジェクトを探す
  walk                   ホストをウォークする
  json                   JSON形式でMIBオブジェクトを表示する
```

### MIBオブジェクトをファイルにダンプする (最初に実行してください)

```
使い方: miburi dump --directory ディレクトリ [--output 出力ファイル名]

Options:
  --directory ディレクトリ, -d ディレクトリ
                         MIBファイルを探索するディレクトリ (複数指定可)
  --output 出力ファイル名, -o 出力ファイル名
                         出力するファイル [デフォルト: smi_objects.gob]
  --help, -h             ヘルプを表示して終了
  --version              バージョンを表示して終了
```

例
```
$ miburi dump -d /usr/share/snmp/mibs -d /usr/share/snmp/mibs/iana -d /usr/share/snmp/mibs/ietf
  ...
Dump completed
```

Miburiは指定のディレクトリから各MIBファイル (MIBファイルは `MIB` をファイル名に含み、非圧縮であること) を解析し、`smi_objects.gob` をカレントディレクトリに作成します。

解析に失敗したMIBファイルは単に無視されます。

### OIDでMIB情報を探す

```
使い方: miburi find [--input 入力ファイル名] --target ターゲット [--verbose]

オプション:
  --input 入力ファイル名, -i 入力ファイル名
                         入力するファイル [デフォルト: smi_objects.gob]
  --target ターゲット, -t ターゲット
                         探すOID (複数指定可)
  --verbose, -v          冗長出力
  --help, -h             ヘルプを表示して終了
  --version              バージョンを表示して終了
```

例:
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

`-t`オプションでは複数のOIDを指定できます。

`-v`オプションを付けると、Type、Unit、Enum、Descriptionも表示します (存在する場合)。

### SNMPデバイスをウォークする

```
使い方: miburi walk [--input 入力ファイル名] [--host ホスト] [--community コミュニティ] [--port ポート] --target ターゲット [--verbose] [--json] [--csv]

オプション:
  --input 入力ファイル名, -i 入力ファイル名
                         入力するファイル [デフォルト: smi_objects.gob]
  --host ホスト, -H ホスト   ウォークするホスト [デフォルト: localhost]
  --community コミュニティ, -c コミュニティ
                         ウォークするコミュニティ [デフォルト: public]
  --port ポート, -p ポート   ウォークするポート [デフォルト: 161]
  --target ターゲット, -t ターゲット
                         ウォークするOID (複数指定可)
  --verbose, -v          冗長出力
  --json, -j             JSON形式で出力
  --csv, -C              CSV形式で出力
  --help, -h             ヘルプを表示して終了
  --version              バージョンを表示して終了
```

SNMP対応デバイスに対してsnmpwalkを実行します。SNMP v2cのみをサポートしています。

指定するOIDの先頭に`1.3.6`が少なくとも必要です。

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

octetstringは文字列値に変換されます。UTF-8文字にならなかった場合は、(hex)が付いた16進数表現で表示されます。

JSONまたはCSV形式で出力することもできます。

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

### MIB情報をJSON形式で表示する

```
使い方: miburi json [--input 入力ファイル名]

オプション:
  --input 入力ファイル名, -i 入力ファイル名
                         ファイルの入力 [デフォルト: smi_objects.gob]
  --help, -h             ヘルプを表示して終了
  --version              バージョンを表示して終了
```

格納されたMIB情報をJSON形式で出力できます。

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

## MIBファイルの取得

主要なMIBファイルを取得するためにDockerを利用できます。

```
$ docker image build -t kmuto/mib:latest .
$ docker container run --rm -v ".:/work" kmuto/mib
```

`mibs` ディレクトリが作成され、MIBファイルが格納されています。これで以下を実行できます:

```
$ miburi dump -d mibs
```

## ライセンス

© 2024 Kenshi Muto

MIT ライセンス ([LICENSE](LICENSE)ファイルを参照)
