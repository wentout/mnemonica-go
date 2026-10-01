package mnemonica_test

import (
	"testing"

	"mnemonica/mnemonica"
)

// Deep-chain READ benchmarks (contract §4.5: construction cost and
// deep-chain read cost at depth 1/10/100 vs a plain object).
//
// Two Go facts shape this file. First, each lineage level needs its own
// declared struct type — Go cannot declare types at runtime, and repeated
// forking cannot deepen a chain — so depth 1/10/100 means 100 declared
// level types, built once here. Second, a promoted field read resolves
// each hop's OFFSET at compile time, but the embedded pointers still
// chase: reading the root's Label through a depth-N instance is N
// dependent memory loads. The benchmarks measure exactly that — and show
// the mnemonica chain costs the same as a plain embedding of identical
// shape. The read closures assert the instance to its level type first;
// that assert is constant across depths and does not affect the
// comparison.

var chainBenchCollection = mnemonica.NewCollection()

type ChainL01 struct {
	mnemonica.Node
	Label string
}

type ChainL02 struct {
	mnemonica.Node
	*ChainL01
}
type ChainL03 struct {
	mnemonica.Node
	*ChainL02
}
type ChainL04 struct {
	mnemonica.Node
	*ChainL03
}
type ChainL05 struct {
	mnemonica.Node
	*ChainL04
}
type ChainL06 struct {
	mnemonica.Node
	*ChainL05
}
type ChainL07 struct {
	mnemonica.Node
	*ChainL06
}
type ChainL08 struct {
	mnemonica.Node
	*ChainL07
}
type ChainL09 struct {
	mnemonica.Node
	*ChainL08
}
type ChainL10 struct {
	mnemonica.Node
	*ChainL09
}
type ChainL11 struct {
	mnemonica.Node
	*ChainL10
}
type ChainL12 struct {
	mnemonica.Node
	*ChainL11
}
type ChainL13 struct {
	mnemonica.Node
	*ChainL12
}
type ChainL14 struct {
	mnemonica.Node
	*ChainL13
}
type ChainL15 struct {
	mnemonica.Node
	*ChainL14
}
type ChainL16 struct {
	mnemonica.Node
	*ChainL15
}
type ChainL17 struct {
	mnemonica.Node
	*ChainL16
}
type ChainL18 struct {
	mnemonica.Node
	*ChainL17
}
type ChainL19 struct {
	mnemonica.Node
	*ChainL18
}
type ChainL20 struct {
	mnemonica.Node
	*ChainL19
}
type ChainL21 struct {
	mnemonica.Node
	*ChainL20
}
type ChainL22 struct {
	mnemonica.Node
	*ChainL21
}
type ChainL23 struct {
	mnemonica.Node
	*ChainL22
}
type ChainL24 struct {
	mnemonica.Node
	*ChainL23
}
type ChainL25 struct {
	mnemonica.Node
	*ChainL24
}
type ChainL26 struct {
	mnemonica.Node
	*ChainL25
}
type ChainL27 struct {
	mnemonica.Node
	*ChainL26
}
type ChainL28 struct {
	mnemonica.Node
	*ChainL27
}
type ChainL29 struct {
	mnemonica.Node
	*ChainL28
}
type ChainL30 struct {
	mnemonica.Node
	*ChainL29
}
type ChainL31 struct {
	mnemonica.Node
	*ChainL30
}
type ChainL32 struct {
	mnemonica.Node
	*ChainL31
}
type ChainL33 struct {
	mnemonica.Node
	*ChainL32
}
type ChainL34 struct {
	mnemonica.Node
	*ChainL33
}
type ChainL35 struct {
	mnemonica.Node
	*ChainL34
}
type ChainL36 struct {
	mnemonica.Node
	*ChainL35
}
type ChainL37 struct {
	mnemonica.Node
	*ChainL36
}
type ChainL38 struct {
	mnemonica.Node
	*ChainL37
}
type ChainL39 struct {
	mnemonica.Node
	*ChainL38
}
type ChainL40 struct {
	mnemonica.Node
	*ChainL39
}
type ChainL41 struct {
	mnemonica.Node
	*ChainL40
}
type ChainL42 struct {
	mnemonica.Node
	*ChainL41
}
type ChainL43 struct {
	mnemonica.Node
	*ChainL42
}
type ChainL44 struct {
	mnemonica.Node
	*ChainL43
}
type ChainL45 struct {
	mnemonica.Node
	*ChainL44
}
type ChainL46 struct {
	mnemonica.Node
	*ChainL45
}
type ChainL47 struct {
	mnemonica.Node
	*ChainL46
}
type ChainL48 struct {
	mnemonica.Node
	*ChainL47
}
type ChainL49 struct {
	mnemonica.Node
	*ChainL48
}
type ChainL50 struct {
	mnemonica.Node
	*ChainL49
}
type ChainL51 struct {
	mnemonica.Node
	*ChainL50
}
type ChainL52 struct {
	mnemonica.Node
	*ChainL51
}
type ChainL53 struct {
	mnemonica.Node
	*ChainL52
}
type ChainL54 struct {
	mnemonica.Node
	*ChainL53
}
type ChainL55 struct {
	mnemonica.Node
	*ChainL54
}
type ChainL56 struct {
	mnemonica.Node
	*ChainL55
}
type ChainL57 struct {
	mnemonica.Node
	*ChainL56
}
type ChainL58 struct {
	mnemonica.Node
	*ChainL57
}
type ChainL59 struct {
	mnemonica.Node
	*ChainL58
}
type ChainL60 struct {
	mnemonica.Node
	*ChainL59
}
type ChainL61 struct {
	mnemonica.Node
	*ChainL60
}
type ChainL62 struct {
	mnemonica.Node
	*ChainL61
}
type ChainL63 struct {
	mnemonica.Node
	*ChainL62
}
type ChainL64 struct {
	mnemonica.Node
	*ChainL63
}
type ChainL65 struct {
	mnemonica.Node
	*ChainL64
}
type ChainL66 struct {
	mnemonica.Node
	*ChainL65
}
type ChainL67 struct {
	mnemonica.Node
	*ChainL66
}
type ChainL68 struct {
	mnemonica.Node
	*ChainL67
}
type ChainL69 struct {
	mnemonica.Node
	*ChainL68
}
type ChainL70 struct {
	mnemonica.Node
	*ChainL69
}
type ChainL71 struct {
	mnemonica.Node
	*ChainL70
}
type ChainL72 struct {
	mnemonica.Node
	*ChainL71
}
type ChainL73 struct {
	mnemonica.Node
	*ChainL72
}
type ChainL74 struct {
	mnemonica.Node
	*ChainL73
}
type ChainL75 struct {
	mnemonica.Node
	*ChainL74
}
type ChainL76 struct {
	mnemonica.Node
	*ChainL75
}
type ChainL77 struct {
	mnemonica.Node
	*ChainL76
}
type ChainL78 struct {
	mnemonica.Node
	*ChainL77
}
type ChainL79 struct {
	mnemonica.Node
	*ChainL78
}
type ChainL80 struct {
	mnemonica.Node
	*ChainL79
}
type ChainL81 struct {
	mnemonica.Node
	*ChainL80
}
type ChainL82 struct {
	mnemonica.Node
	*ChainL81
}
type ChainL83 struct {
	mnemonica.Node
	*ChainL82
}
type ChainL84 struct {
	mnemonica.Node
	*ChainL83
}
type ChainL85 struct {
	mnemonica.Node
	*ChainL84
}
type ChainL86 struct {
	mnemonica.Node
	*ChainL85
}
type ChainL87 struct {
	mnemonica.Node
	*ChainL86
}
type ChainL88 struct {
	mnemonica.Node
	*ChainL87
}
type ChainL89 struct {
	mnemonica.Node
	*ChainL88
}
type ChainL90 struct {
	mnemonica.Node
	*ChainL89
}
type ChainL91 struct {
	mnemonica.Node
	*ChainL90
}
type ChainL92 struct {
	mnemonica.Node
	*ChainL91
}
type ChainL93 struct {
	mnemonica.Node
	*ChainL92
}
type ChainL94 struct {
	mnemonica.Node
	*ChainL93
}
type ChainL95 struct {
	mnemonica.Node
	*ChainL94
}
type ChainL96 struct {
	mnemonica.Node
	*ChainL95
}
type ChainL97 struct {
	mnemonica.Node
	*ChainL96
}
type ChainL98 struct {
	mnemonica.Node
	*ChainL97
}
type ChainL99 struct {
	mnemonica.Node
	*ChainL98
}
type ChainL100 struct {
	mnemonica.Node
	*ChainL99
}

var ChainL01T = mnemonica.Must(mnemonica.Define[ChainL01](chainBenchCollection, "ChainL01", func(c *ChainL01, label string) error {
	c.Label = label
	return nil
}))

var ChainL02T = mnemonica.Must(mnemonica.Sub[ChainL02](ChainL01T, "ChainL02", func(*ChainL02, string) error { return nil }))
var ChainL03T = mnemonica.Must(mnemonica.Sub[ChainL03](ChainL02T, "ChainL03", func(*ChainL03, string) error { return nil }))
var ChainL04T = mnemonica.Must(mnemonica.Sub[ChainL04](ChainL03T, "ChainL04", func(*ChainL04, string) error { return nil }))
var ChainL05T = mnemonica.Must(mnemonica.Sub[ChainL05](ChainL04T, "ChainL05", func(*ChainL05, string) error { return nil }))
var ChainL06T = mnemonica.Must(mnemonica.Sub[ChainL06](ChainL05T, "ChainL06", func(*ChainL06, string) error { return nil }))
var ChainL07T = mnemonica.Must(mnemonica.Sub[ChainL07](ChainL06T, "ChainL07", func(*ChainL07, string) error { return nil }))
var ChainL08T = mnemonica.Must(mnemonica.Sub[ChainL08](ChainL07T, "ChainL08", func(*ChainL08, string) error { return nil }))
var ChainL09T = mnemonica.Must(mnemonica.Sub[ChainL09](ChainL08T, "ChainL09", func(*ChainL09, string) error { return nil }))
var ChainL10T = mnemonica.Must(mnemonica.Sub[ChainL10](ChainL09T, "ChainL10", func(*ChainL10, string) error { return nil }))
var ChainL11T = mnemonica.Must(mnemonica.Sub[ChainL11](ChainL10T, "ChainL11", func(*ChainL11, string) error { return nil }))
var ChainL12T = mnemonica.Must(mnemonica.Sub[ChainL12](ChainL11T, "ChainL12", func(*ChainL12, string) error { return nil }))
var ChainL13T = mnemonica.Must(mnemonica.Sub[ChainL13](ChainL12T, "ChainL13", func(*ChainL13, string) error { return nil }))
var ChainL14T = mnemonica.Must(mnemonica.Sub[ChainL14](ChainL13T, "ChainL14", func(*ChainL14, string) error { return nil }))
var ChainL15T = mnemonica.Must(mnemonica.Sub[ChainL15](ChainL14T, "ChainL15", func(*ChainL15, string) error { return nil }))
var ChainL16T = mnemonica.Must(mnemonica.Sub[ChainL16](ChainL15T, "ChainL16", func(*ChainL16, string) error { return nil }))
var ChainL17T = mnemonica.Must(mnemonica.Sub[ChainL17](ChainL16T, "ChainL17", func(*ChainL17, string) error { return nil }))
var ChainL18T = mnemonica.Must(mnemonica.Sub[ChainL18](ChainL17T, "ChainL18", func(*ChainL18, string) error { return nil }))
var ChainL19T = mnemonica.Must(mnemonica.Sub[ChainL19](ChainL18T, "ChainL19", func(*ChainL19, string) error { return nil }))
var ChainL20T = mnemonica.Must(mnemonica.Sub[ChainL20](ChainL19T, "ChainL20", func(*ChainL20, string) error { return nil }))
var ChainL21T = mnemonica.Must(mnemonica.Sub[ChainL21](ChainL20T, "ChainL21", func(*ChainL21, string) error { return nil }))
var ChainL22T = mnemonica.Must(mnemonica.Sub[ChainL22](ChainL21T, "ChainL22", func(*ChainL22, string) error { return nil }))
var ChainL23T = mnemonica.Must(mnemonica.Sub[ChainL23](ChainL22T, "ChainL23", func(*ChainL23, string) error { return nil }))
var ChainL24T = mnemonica.Must(mnemonica.Sub[ChainL24](ChainL23T, "ChainL24", func(*ChainL24, string) error { return nil }))
var ChainL25T = mnemonica.Must(mnemonica.Sub[ChainL25](ChainL24T, "ChainL25", func(*ChainL25, string) error { return nil }))
var ChainL26T = mnemonica.Must(mnemonica.Sub[ChainL26](ChainL25T, "ChainL26", func(*ChainL26, string) error { return nil }))
var ChainL27T = mnemonica.Must(mnemonica.Sub[ChainL27](ChainL26T, "ChainL27", func(*ChainL27, string) error { return nil }))
var ChainL28T = mnemonica.Must(mnemonica.Sub[ChainL28](ChainL27T, "ChainL28", func(*ChainL28, string) error { return nil }))
var ChainL29T = mnemonica.Must(mnemonica.Sub[ChainL29](ChainL28T, "ChainL29", func(*ChainL29, string) error { return nil }))
var ChainL30T = mnemonica.Must(mnemonica.Sub[ChainL30](ChainL29T, "ChainL30", func(*ChainL30, string) error { return nil }))
var ChainL31T = mnemonica.Must(mnemonica.Sub[ChainL31](ChainL30T, "ChainL31", func(*ChainL31, string) error { return nil }))
var ChainL32T = mnemonica.Must(mnemonica.Sub[ChainL32](ChainL31T, "ChainL32", func(*ChainL32, string) error { return nil }))
var ChainL33T = mnemonica.Must(mnemonica.Sub[ChainL33](ChainL32T, "ChainL33", func(*ChainL33, string) error { return nil }))
var ChainL34T = mnemonica.Must(mnemonica.Sub[ChainL34](ChainL33T, "ChainL34", func(*ChainL34, string) error { return nil }))
var ChainL35T = mnemonica.Must(mnemonica.Sub[ChainL35](ChainL34T, "ChainL35", func(*ChainL35, string) error { return nil }))
var ChainL36T = mnemonica.Must(mnemonica.Sub[ChainL36](ChainL35T, "ChainL36", func(*ChainL36, string) error { return nil }))
var ChainL37T = mnemonica.Must(mnemonica.Sub[ChainL37](ChainL36T, "ChainL37", func(*ChainL37, string) error { return nil }))
var ChainL38T = mnemonica.Must(mnemonica.Sub[ChainL38](ChainL37T, "ChainL38", func(*ChainL38, string) error { return nil }))
var ChainL39T = mnemonica.Must(mnemonica.Sub[ChainL39](ChainL38T, "ChainL39", func(*ChainL39, string) error { return nil }))
var ChainL40T = mnemonica.Must(mnemonica.Sub[ChainL40](ChainL39T, "ChainL40", func(*ChainL40, string) error { return nil }))
var ChainL41T = mnemonica.Must(mnemonica.Sub[ChainL41](ChainL40T, "ChainL41", func(*ChainL41, string) error { return nil }))
var ChainL42T = mnemonica.Must(mnemonica.Sub[ChainL42](ChainL41T, "ChainL42", func(*ChainL42, string) error { return nil }))
var ChainL43T = mnemonica.Must(mnemonica.Sub[ChainL43](ChainL42T, "ChainL43", func(*ChainL43, string) error { return nil }))
var ChainL44T = mnemonica.Must(mnemonica.Sub[ChainL44](ChainL43T, "ChainL44", func(*ChainL44, string) error { return nil }))
var ChainL45T = mnemonica.Must(mnemonica.Sub[ChainL45](ChainL44T, "ChainL45", func(*ChainL45, string) error { return nil }))
var ChainL46T = mnemonica.Must(mnemonica.Sub[ChainL46](ChainL45T, "ChainL46", func(*ChainL46, string) error { return nil }))
var ChainL47T = mnemonica.Must(mnemonica.Sub[ChainL47](ChainL46T, "ChainL47", func(*ChainL47, string) error { return nil }))
var ChainL48T = mnemonica.Must(mnemonica.Sub[ChainL48](ChainL47T, "ChainL48", func(*ChainL48, string) error { return nil }))
var ChainL49T = mnemonica.Must(mnemonica.Sub[ChainL49](ChainL48T, "ChainL49", func(*ChainL49, string) error { return nil }))
var ChainL50T = mnemonica.Must(mnemonica.Sub[ChainL50](ChainL49T, "ChainL50", func(*ChainL50, string) error { return nil }))
var ChainL51T = mnemonica.Must(mnemonica.Sub[ChainL51](ChainL50T, "ChainL51", func(*ChainL51, string) error { return nil }))
var ChainL52T = mnemonica.Must(mnemonica.Sub[ChainL52](ChainL51T, "ChainL52", func(*ChainL52, string) error { return nil }))
var ChainL53T = mnemonica.Must(mnemonica.Sub[ChainL53](ChainL52T, "ChainL53", func(*ChainL53, string) error { return nil }))
var ChainL54T = mnemonica.Must(mnemonica.Sub[ChainL54](ChainL53T, "ChainL54", func(*ChainL54, string) error { return nil }))
var ChainL55T = mnemonica.Must(mnemonica.Sub[ChainL55](ChainL54T, "ChainL55", func(*ChainL55, string) error { return nil }))
var ChainL56T = mnemonica.Must(mnemonica.Sub[ChainL56](ChainL55T, "ChainL56", func(*ChainL56, string) error { return nil }))
var ChainL57T = mnemonica.Must(mnemonica.Sub[ChainL57](ChainL56T, "ChainL57", func(*ChainL57, string) error { return nil }))
var ChainL58T = mnemonica.Must(mnemonica.Sub[ChainL58](ChainL57T, "ChainL58", func(*ChainL58, string) error { return nil }))
var ChainL59T = mnemonica.Must(mnemonica.Sub[ChainL59](ChainL58T, "ChainL59", func(*ChainL59, string) error { return nil }))
var ChainL60T = mnemonica.Must(mnemonica.Sub[ChainL60](ChainL59T, "ChainL60", func(*ChainL60, string) error { return nil }))
var ChainL61T = mnemonica.Must(mnemonica.Sub[ChainL61](ChainL60T, "ChainL61", func(*ChainL61, string) error { return nil }))
var ChainL62T = mnemonica.Must(mnemonica.Sub[ChainL62](ChainL61T, "ChainL62", func(*ChainL62, string) error { return nil }))
var ChainL63T = mnemonica.Must(mnemonica.Sub[ChainL63](ChainL62T, "ChainL63", func(*ChainL63, string) error { return nil }))
var ChainL64T = mnemonica.Must(mnemonica.Sub[ChainL64](ChainL63T, "ChainL64", func(*ChainL64, string) error { return nil }))
var ChainL65T = mnemonica.Must(mnemonica.Sub[ChainL65](ChainL64T, "ChainL65", func(*ChainL65, string) error { return nil }))
var ChainL66T = mnemonica.Must(mnemonica.Sub[ChainL66](ChainL65T, "ChainL66", func(*ChainL66, string) error { return nil }))
var ChainL67T = mnemonica.Must(mnemonica.Sub[ChainL67](ChainL66T, "ChainL67", func(*ChainL67, string) error { return nil }))
var ChainL68T = mnemonica.Must(mnemonica.Sub[ChainL68](ChainL67T, "ChainL68", func(*ChainL68, string) error { return nil }))
var ChainL69T = mnemonica.Must(mnemonica.Sub[ChainL69](ChainL68T, "ChainL69", func(*ChainL69, string) error { return nil }))
var ChainL70T = mnemonica.Must(mnemonica.Sub[ChainL70](ChainL69T, "ChainL70", func(*ChainL70, string) error { return nil }))
var ChainL71T = mnemonica.Must(mnemonica.Sub[ChainL71](ChainL70T, "ChainL71", func(*ChainL71, string) error { return nil }))
var ChainL72T = mnemonica.Must(mnemonica.Sub[ChainL72](ChainL71T, "ChainL72", func(*ChainL72, string) error { return nil }))
var ChainL73T = mnemonica.Must(mnemonica.Sub[ChainL73](ChainL72T, "ChainL73", func(*ChainL73, string) error { return nil }))
var ChainL74T = mnemonica.Must(mnemonica.Sub[ChainL74](ChainL73T, "ChainL74", func(*ChainL74, string) error { return nil }))
var ChainL75T = mnemonica.Must(mnemonica.Sub[ChainL75](ChainL74T, "ChainL75", func(*ChainL75, string) error { return nil }))
var ChainL76T = mnemonica.Must(mnemonica.Sub[ChainL76](ChainL75T, "ChainL76", func(*ChainL76, string) error { return nil }))
var ChainL77T = mnemonica.Must(mnemonica.Sub[ChainL77](ChainL76T, "ChainL77", func(*ChainL77, string) error { return nil }))
var ChainL78T = mnemonica.Must(mnemonica.Sub[ChainL78](ChainL77T, "ChainL78", func(*ChainL78, string) error { return nil }))
var ChainL79T = mnemonica.Must(mnemonica.Sub[ChainL79](ChainL78T, "ChainL79", func(*ChainL79, string) error { return nil }))
var ChainL80T = mnemonica.Must(mnemonica.Sub[ChainL80](ChainL79T, "ChainL80", func(*ChainL80, string) error { return nil }))
var ChainL81T = mnemonica.Must(mnemonica.Sub[ChainL81](ChainL80T, "ChainL81", func(*ChainL81, string) error { return nil }))
var ChainL82T = mnemonica.Must(mnemonica.Sub[ChainL82](ChainL81T, "ChainL82", func(*ChainL82, string) error { return nil }))
var ChainL83T = mnemonica.Must(mnemonica.Sub[ChainL83](ChainL82T, "ChainL83", func(*ChainL83, string) error { return nil }))
var ChainL84T = mnemonica.Must(mnemonica.Sub[ChainL84](ChainL83T, "ChainL84", func(*ChainL84, string) error { return nil }))
var ChainL85T = mnemonica.Must(mnemonica.Sub[ChainL85](ChainL84T, "ChainL85", func(*ChainL85, string) error { return nil }))
var ChainL86T = mnemonica.Must(mnemonica.Sub[ChainL86](ChainL85T, "ChainL86", func(*ChainL86, string) error { return nil }))
var ChainL87T = mnemonica.Must(mnemonica.Sub[ChainL87](ChainL86T, "ChainL87", func(*ChainL87, string) error { return nil }))
var ChainL88T = mnemonica.Must(mnemonica.Sub[ChainL88](ChainL87T, "ChainL88", func(*ChainL88, string) error { return nil }))
var ChainL89T = mnemonica.Must(mnemonica.Sub[ChainL89](ChainL88T, "ChainL89", func(*ChainL89, string) error { return nil }))
var ChainL90T = mnemonica.Must(mnemonica.Sub[ChainL90](ChainL89T, "ChainL90", func(*ChainL90, string) error { return nil }))
var ChainL91T = mnemonica.Must(mnemonica.Sub[ChainL91](ChainL90T, "ChainL91", func(*ChainL91, string) error { return nil }))
var ChainL92T = mnemonica.Must(mnemonica.Sub[ChainL92](ChainL91T, "ChainL92", func(*ChainL92, string) error { return nil }))
var ChainL93T = mnemonica.Must(mnemonica.Sub[ChainL93](ChainL92T, "ChainL93", func(*ChainL93, string) error { return nil }))
var ChainL94T = mnemonica.Must(mnemonica.Sub[ChainL94](ChainL93T, "ChainL94", func(*ChainL94, string) error { return nil }))
var ChainL95T = mnemonica.Must(mnemonica.Sub[ChainL95](ChainL94T, "ChainL95", func(*ChainL95, string) error { return nil }))
var ChainL96T = mnemonica.Must(mnemonica.Sub[ChainL96](ChainL95T, "ChainL96", func(*ChainL96, string) error { return nil }))
var ChainL97T = mnemonica.Must(mnemonica.Sub[ChainL97](ChainL96T, "ChainL97", func(*ChainL97, string) error { return nil }))
var ChainL98T = mnemonica.Must(mnemonica.Sub[ChainL98](ChainL97T, "ChainL98", func(*ChainL98, string) error { return nil }))
var ChainL99T = mnemonica.Must(mnemonica.Sub[ChainL99](ChainL98T, "ChainL99", func(*ChainL99, string) error { return nil }))
var ChainL100T = mnemonica.Must(mnemonica.Sub[ChainL100](ChainL99T, "ChainL100", func(*ChainL100, string) error { return nil }))

// chainFrom[level] constructs the given level FROM the level-1 instance
// held in the first argument — the untyped bridge, so one table serves
// every depth.
var chainFrom = []func(mnemonica.Instance, any) (mnemonica.Instance, error){
	2:   ChainL02T.Type.From,
	3:   ChainL03T.Type.From,
	4:   ChainL04T.Type.From,
	5:   ChainL05T.Type.From,
	6:   ChainL06T.Type.From,
	7:   ChainL07T.Type.From,
	8:   ChainL08T.Type.From,
	9:   ChainL09T.Type.From,
	10:  ChainL10T.Type.From,
	11:  ChainL11T.Type.From,
	12:  ChainL12T.Type.From,
	13:  ChainL13T.Type.From,
	14:  ChainL14T.Type.From,
	15:  ChainL15T.Type.From,
	16:  ChainL16T.Type.From,
	17:  ChainL17T.Type.From,
	18:  ChainL18T.Type.From,
	19:  ChainL19T.Type.From,
	20:  ChainL20T.Type.From,
	21:  ChainL21T.Type.From,
	22:  ChainL22T.Type.From,
	23:  ChainL23T.Type.From,
	24:  ChainL24T.Type.From,
	25:  ChainL25T.Type.From,
	26:  ChainL26T.Type.From,
	27:  ChainL27T.Type.From,
	28:  ChainL28T.Type.From,
	29:  ChainL29T.Type.From,
	30:  ChainL30T.Type.From,
	31:  ChainL31T.Type.From,
	32:  ChainL32T.Type.From,
	33:  ChainL33T.Type.From,
	34:  ChainL34T.Type.From,
	35:  ChainL35T.Type.From,
	36:  ChainL36T.Type.From,
	37:  ChainL37T.Type.From,
	38:  ChainL38T.Type.From,
	39:  ChainL39T.Type.From,
	40:  ChainL40T.Type.From,
	41:  ChainL41T.Type.From,
	42:  ChainL42T.Type.From,
	43:  ChainL43T.Type.From,
	44:  ChainL44T.Type.From,
	45:  ChainL45T.Type.From,
	46:  ChainL46T.Type.From,
	47:  ChainL47T.Type.From,
	48:  ChainL48T.Type.From,
	49:  ChainL49T.Type.From,
	50:  ChainL50T.Type.From,
	51:  ChainL51T.Type.From,
	52:  ChainL52T.Type.From,
	53:  ChainL53T.Type.From,
	54:  ChainL54T.Type.From,
	55:  ChainL55T.Type.From,
	56:  ChainL56T.Type.From,
	57:  ChainL57T.Type.From,
	58:  ChainL58T.Type.From,
	59:  ChainL59T.Type.From,
	60:  ChainL60T.Type.From,
	61:  ChainL61T.Type.From,
	62:  ChainL62T.Type.From,
	63:  ChainL63T.Type.From,
	64:  ChainL64T.Type.From,
	65:  ChainL65T.Type.From,
	66:  ChainL66T.Type.From,
	67:  ChainL67T.Type.From,
	68:  ChainL68T.Type.From,
	69:  ChainL69T.Type.From,
	70:  ChainL70T.Type.From,
	71:  ChainL71T.Type.From,
	72:  ChainL72T.Type.From,
	73:  ChainL73T.Type.From,
	74:  ChainL74T.Type.From,
	75:  ChainL75T.Type.From,
	76:  ChainL76T.Type.From,
	77:  ChainL77T.Type.From,
	78:  ChainL78T.Type.From,
	79:  ChainL79T.Type.From,
	80:  ChainL80T.Type.From,
	81:  ChainL81T.Type.From,
	82:  ChainL82T.Type.From,
	83:  ChainL83T.Type.From,
	84:  ChainL84T.Type.From,
	85:  ChainL85T.Type.From,
	86:  ChainL86T.Type.From,
	87:  ChainL87T.Type.From,
	88:  ChainL88T.Type.From,
	89:  ChainL89T.Type.From,
	90:  ChainL90T.Type.From,
	91:  ChainL91T.Type.From,
	92:  ChainL92T.Type.From,
	93:  ChainL93T.Type.From,
	94:  ChainL94T.Type.From,
	95:  ChainL95T.Type.From,
	96:  ChainL96T.Type.From,
	97:  ChainL97T.Type.From,
	98:  ChainL98T.Type.From,
	99:  ChainL99T.Type.From,
	100: ChainL100T.Type.From,
}

func chainOrPanic[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}

// buildChain constructs the mnemonica chain of the given depth (1..100).
// Construction happens outside the timed region in every benchmark.
func buildChain(depth int, label string) mnemonica.Instance {
	current := mnemonica.Instance(chainOrPanic(ChainL01T.New(label)))
	for level := 2; level <= depth; level++ {
		next, err := chainFrom[level](current, label)
		if err != nil {
			panic(err)
		}
		current = next
	}
	return current
}

// benchPromotedRead times read(), which must return the root's Label.
func benchPromotedRead(b *testing.B, read func() string) {
	b.Helper()
	if got := read(); got != "deep-label" {
		b.Fatalf("promoted read = %q, want the root label", got)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchSink = read()
	}
}

func BenchmarkChainReadDepth1(b *testing.B) {
	leaf := buildChain(1, "deep-label")
	benchPromotedRead(b, func() string { return leaf.(*ChainL01).Label })
}

func BenchmarkChainReadDepth10(b *testing.B) {
	leaf := buildChain(10, "deep-label")
	benchPromotedRead(b, func() string { return leaf.(*ChainL10).Label })
}

func BenchmarkChainReadDepth100(b *testing.B) {
	leaf := buildChain(100, "deep-label")
	benchPromotedRead(b, func() string { return leaf.(*ChainL100).Label })
}

// The plain comparison chains: identical shape without mnemonica — no
// Node headers, no lineage, just embedding. Every declared type is
// referenced by the constructors table below (the reflect setter path is
// what the mnemonica chain already benchmarks elsewhere; here the build
// is one-time, outside the timed region).

type PlainP01 struct{ Label string }
type PlainP02 struct{ *PlainP01 }
type PlainP03 struct{ *PlainP02 }
type PlainP04 struct{ *PlainP03 }
type PlainP05 struct{ *PlainP04 }
type PlainP06 struct{ *PlainP05 }
type PlainP07 struct{ *PlainP06 }
type PlainP08 struct{ *PlainP07 }
type PlainP09 struct{ *PlainP08 }
type PlainP10 struct{ *PlainP09 }
type PlainP11 struct{ *PlainP10 }
type PlainP12 struct{ *PlainP11 }
type PlainP13 struct{ *PlainP12 }
type PlainP14 struct{ *PlainP13 }
type PlainP15 struct{ *PlainP14 }
type PlainP16 struct{ *PlainP15 }
type PlainP17 struct{ *PlainP16 }
type PlainP18 struct{ *PlainP17 }
type PlainP19 struct{ *PlainP18 }
type PlainP20 struct{ *PlainP19 }
type PlainP21 struct{ *PlainP20 }
type PlainP22 struct{ *PlainP21 }
type PlainP23 struct{ *PlainP22 }
type PlainP24 struct{ *PlainP23 }
type PlainP25 struct{ *PlainP24 }
type PlainP26 struct{ *PlainP25 }
type PlainP27 struct{ *PlainP26 }
type PlainP28 struct{ *PlainP27 }
type PlainP29 struct{ *PlainP28 }
type PlainP30 struct{ *PlainP29 }
type PlainP31 struct{ *PlainP30 }
type PlainP32 struct{ *PlainP31 }
type PlainP33 struct{ *PlainP32 }
type PlainP34 struct{ *PlainP33 }
type PlainP35 struct{ *PlainP34 }
type PlainP36 struct{ *PlainP35 }
type PlainP37 struct{ *PlainP36 }
type PlainP38 struct{ *PlainP37 }
type PlainP39 struct{ *PlainP38 }
type PlainP40 struct{ *PlainP39 }
type PlainP41 struct{ *PlainP40 }
type PlainP42 struct{ *PlainP41 }
type PlainP43 struct{ *PlainP42 }
type PlainP44 struct{ *PlainP43 }
type PlainP45 struct{ *PlainP44 }
type PlainP46 struct{ *PlainP45 }
type PlainP47 struct{ *PlainP46 }
type PlainP48 struct{ *PlainP47 }
type PlainP49 struct{ *PlainP48 }
type PlainP50 struct{ *PlainP49 }
type PlainP51 struct{ *PlainP50 }
type PlainP52 struct{ *PlainP51 }
type PlainP53 struct{ *PlainP52 }
type PlainP54 struct{ *PlainP53 }
type PlainP55 struct{ *PlainP54 }
type PlainP56 struct{ *PlainP55 }
type PlainP57 struct{ *PlainP56 }
type PlainP58 struct{ *PlainP57 }
type PlainP59 struct{ *PlainP58 }
type PlainP60 struct{ *PlainP59 }
type PlainP61 struct{ *PlainP60 }
type PlainP62 struct{ *PlainP61 }
type PlainP63 struct{ *PlainP62 }
type PlainP64 struct{ *PlainP63 }
type PlainP65 struct{ *PlainP64 }
type PlainP66 struct{ *PlainP65 }
type PlainP67 struct{ *PlainP66 }
type PlainP68 struct{ *PlainP67 }
type PlainP69 struct{ *PlainP68 }
type PlainP70 struct{ *PlainP69 }
type PlainP71 struct{ *PlainP70 }
type PlainP72 struct{ *PlainP71 }
type PlainP73 struct{ *PlainP72 }
type PlainP74 struct{ *PlainP73 }
type PlainP75 struct{ *PlainP74 }
type PlainP76 struct{ *PlainP75 }
type PlainP77 struct{ *PlainP76 }
type PlainP78 struct{ *PlainP77 }
type PlainP79 struct{ *PlainP78 }
type PlainP80 struct{ *PlainP79 }
type PlainP81 struct{ *PlainP80 }
type PlainP82 struct{ *PlainP81 }
type PlainP83 struct{ *PlainP82 }
type PlainP84 struct{ *PlainP83 }
type PlainP85 struct{ *PlainP84 }
type PlainP86 struct{ *PlainP85 }
type PlainP87 struct{ *PlainP86 }
type PlainP88 struct{ *PlainP87 }
type PlainP89 struct{ *PlainP88 }
type PlainP90 struct{ *PlainP89 }
type PlainP91 struct{ *PlainP90 }
type PlainP92 struct{ *PlainP91 }
type PlainP93 struct{ *PlainP92 }
type PlainP94 struct{ *PlainP93 }
type PlainP95 struct{ *PlainP94 }
type PlainP96 struct{ *PlainP95 }
type PlainP97 struct{ *PlainP96 }
type PlainP98 struct{ *PlainP97 }
type PlainP99 struct{ *PlainP98 }
type PlainP100 struct{ *PlainP99 }

var plainFrom = []func(parent any) any{
	2:   func(p any) any { return &PlainP02{p.(*PlainP01)} },
	3:   func(p any) any { return &PlainP03{p.(*PlainP02)} },
	4:   func(p any) any { return &PlainP04{p.(*PlainP03)} },
	5:   func(p any) any { return &PlainP05{p.(*PlainP04)} },
	6:   func(p any) any { return &PlainP06{p.(*PlainP05)} },
	7:   func(p any) any { return &PlainP07{p.(*PlainP06)} },
	8:   func(p any) any { return &PlainP08{p.(*PlainP07)} },
	9:   func(p any) any { return &PlainP09{p.(*PlainP08)} },
	10:  func(p any) any { return &PlainP10{p.(*PlainP09)} },
	11:  func(p any) any { return &PlainP11{p.(*PlainP10)} },
	12:  func(p any) any { return &PlainP12{p.(*PlainP11)} },
	13:  func(p any) any { return &PlainP13{p.(*PlainP12)} },
	14:  func(p any) any { return &PlainP14{p.(*PlainP13)} },
	15:  func(p any) any { return &PlainP15{p.(*PlainP14)} },
	16:  func(p any) any { return &PlainP16{p.(*PlainP15)} },
	17:  func(p any) any { return &PlainP17{p.(*PlainP16)} },
	18:  func(p any) any { return &PlainP18{p.(*PlainP17)} },
	19:  func(p any) any { return &PlainP19{p.(*PlainP18)} },
	20:  func(p any) any { return &PlainP20{p.(*PlainP19)} },
	21:  func(p any) any { return &PlainP21{p.(*PlainP20)} },
	22:  func(p any) any { return &PlainP22{p.(*PlainP21)} },
	23:  func(p any) any { return &PlainP23{p.(*PlainP22)} },
	24:  func(p any) any { return &PlainP24{p.(*PlainP23)} },
	25:  func(p any) any { return &PlainP25{p.(*PlainP24)} },
	26:  func(p any) any { return &PlainP26{p.(*PlainP25)} },
	27:  func(p any) any { return &PlainP27{p.(*PlainP26)} },
	28:  func(p any) any { return &PlainP28{p.(*PlainP27)} },
	29:  func(p any) any { return &PlainP29{p.(*PlainP28)} },
	30:  func(p any) any { return &PlainP30{p.(*PlainP29)} },
	31:  func(p any) any { return &PlainP31{p.(*PlainP30)} },
	32:  func(p any) any { return &PlainP32{p.(*PlainP31)} },
	33:  func(p any) any { return &PlainP33{p.(*PlainP32)} },
	34:  func(p any) any { return &PlainP34{p.(*PlainP33)} },
	35:  func(p any) any { return &PlainP35{p.(*PlainP34)} },
	36:  func(p any) any { return &PlainP36{p.(*PlainP35)} },
	37:  func(p any) any { return &PlainP37{p.(*PlainP36)} },
	38:  func(p any) any { return &PlainP38{p.(*PlainP37)} },
	39:  func(p any) any { return &PlainP39{p.(*PlainP38)} },
	40:  func(p any) any { return &PlainP40{p.(*PlainP39)} },
	41:  func(p any) any { return &PlainP41{p.(*PlainP40)} },
	42:  func(p any) any { return &PlainP42{p.(*PlainP41)} },
	43:  func(p any) any { return &PlainP43{p.(*PlainP42)} },
	44:  func(p any) any { return &PlainP44{p.(*PlainP43)} },
	45:  func(p any) any { return &PlainP45{p.(*PlainP44)} },
	46:  func(p any) any { return &PlainP46{p.(*PlainP45)} },
	47:  func(p any) any { return &PlainP47{p.(*PlainP46)} },
	48:  func(p any) any { return &PlainP48{p.(*PlainP47)} },
	49:  func(p any) any { return &PlainP49{p.(*PlainP48)} },
	50:  func(p any) any { return &PlainP50{p.(*PlainP49)} },
	51:  func(p any) any { return &PlainP51{p.(*PlainP50)} },
	52:  func(p any) any { return &PlainP52{p.(*PlainP51)} },
	53:  func(p any) any { return &PlainP53{p.(*PlainP52)} },
	54:  func(p any) any { return &PlainP54{p.(*PlainP53)} },
	55:  func(p any) any { return &PlainP55{p.(*PlainP54)} },
	56:  func(p any) any { return &PlainP56{p.(*PlainP55)} },
	57:  func(p any) any { return &PlainP57{p.(*PlainP56)} },
	58:  func(p any) any { return &PlainP58{p.(*PlainP57)} },
	59:  func(p any) any { return &PlainP59{p.(*PlainP58)} },
	60:  func(p any) any { return &PlainP60{p.(*PlainP59)} },
	61:  func(p any) any { return &PlainP61{p.(*PlainP60)} },
	62:  func(p any) any { return &PlainP62{p.(*PlainP61)} },
	63:  func(p any) any { return &PlainP63{p.(*PlainP62)} },
	64:  func(p any) any { return &PlainP64{p.(*PlainP63)} },
	65:  func(p any) any { return &PlainP65{p.(*PlainP64)} },
	66:  func(p any) any { return &PlainP66{p.(*PlainP65)} },
	67:  func(p any) any { return &PlainP67{p.(*PlainP66)} },
	68:  func(p any) any { return &PlainP68{p.(*PlainP67)} },
	69:  func(p any) any { return &PlainP69{p.(*PlainP68)} },
	70:  func(p any) any { return &PlainP70{p.(*PlainP69)} },
	71:  func(p any) any { return &PlainP71{p.(*PlainP70)} },
	72:  func(p any) any { return &PlainP72{p.(*PlainP71)} },
	73:  func(p any) any { return &PlainP73{p.(*PlainP72)} },
	74:  func(p any) any { return &PlainP74{p.(*PlainP73)} },
	75:  func(p any) any { return &PlainP75{p.(*PlainP74)} },
	76:  func(p any) any { return &PlainP76{p.(*PlainP75)} },
	77:  func(p any) any { return &PlainP77{p.(*PlainP76)} },
	78:  func(p any) any { return &PlainP78{p.(*PlainP77)} },
	79:  func(p any) any { return &PlainP79{p.(*PlainP78)} },
	80:  func(p any) any { return &PlainP80{p.(*PlainP79)} },
	81:  func(p any) any { return &PlainP81{p.(*PlainP80)} },
	82:  func(p any) any { return &PlainP82{p.(*PlainP81)} },
	83:  func(p any) any { return &PlainP83{p.(*PlainP82)} },
	84:  func(p any) any { return &PlainP84{p.(*PlainP83)} },
	85:  func(p any) any { return &PlainP85{p.(*PlainP84)} },
	86:  func(p any) any { return &PlainP86{p.(*PlainP85)} },
	87:  func(p any) any { return &PlainP87{p.(*PlainP86)} },
	88:  func(p any) any { return &PlainP88{p.(*PlainP87)} },
	89:  func(p any) any { return &PlainP89{p.(*PlainP88)} },
	90:  func(p any) any { return &PlainP90{p.(*PlainP89)} },
	91:  func(p any) any { return &PlainP91{p.(*PlainP90)} },
	92:  func(p any) any { return &PlainP92{p.(*PlainP91)} },
	93:  func(p any) any { return &PlainP93{p.(*PlainP92)} },
	94:  func(p any) any { return &PlainP94{p.(*PlainP93)} },
	95:  func(p any) any { return &PlainP95{p.(*PlainP94)} },
	96:  func(p any) any { return &PlainP96{p.(*PlainP95)} },
	97:  func(p any) any { return &PlainP97{p.(*PlainP96)} },
	98:  func(p any) any { return &PlainP98{p.(*PlainP97)} },
	99:  func(p any) any { return &PlainP99{p.(*PlainP98)} },
	100: func(p any) any { return &PlainP100{p.(*PlainP99)} },
}

func buildPlainChain(depth int) any {
	current := any(&PlainP01{Label: "deep-label"})
	for level := 2; level <= depth; level++ {
		current = plainFrom[level](current)
	}
	return current
}

var plainDepth1 = buildPlainChain(1)
var plainDepth10 = buildPlainChain(10)
var plainDepth100 = buildPlainChain(100)

func BenchmarkPlainReadDepth1(b *testing.B) {
	leaf := plainDepth1.(*PlainP01)
	benchPromotedRead(b, func() string { return leaf.Label })
}

func BenchmarkPlainReadDepth10(b *testing.B) {
	leaf := plainDepth10.(*PlainP10)
	benchPromotedRead(b, func() string { return leaf.Label })
}

func BenchmarkPlainReadDepth100(b *testing.B) {
	leaf := plainDepth100.(*PlainP100)
	benchPromotedRead(b, func() string { return leaf.Label })
}
