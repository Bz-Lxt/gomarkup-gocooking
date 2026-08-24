package engine

import (
	"math"
	"strconv"
	"strings"
	"unicode"

	"gocooking/internal/model"
)

func formatFloat(v float64) string {
	s := strconv.FormatFloat(v, 'f', 3, 64)
	s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	if s == "-0" || s == "" {
		return "0"
	}
	return s
}

// FloatTol 是单位换算验收基线（< 1e-6）。
// 只对有明确定义的换算系数生效：
//
//	重量：g=1, kg/千克/公斤=1000, 斤=500, 两=50
//	容积：ml/毫升=1, L/l/升=1000
//	计数：个/只/枚 视为 1:1（都归一到「个」）
//	无量纲：适量/少许/若干 → 显示「适量」，数量不累加
//	其余（把/根/片/块/瓣/颗/条/碗/杯）保持原单位，绝不编造系数
const FloatTol = 1e-6

// Dimension 量纲。unknown 表示只能与完全相同单位合并。
type Dimension string

const (
	DimWeight        Dimension = model.DimWeight
	DimVolume        Dimension = model.DimVolume
	DimCount         Dimension = model.DimCount
	DimDimensionless Dimension = model.DimDimensionless
	DimUnknown       Dimension = model.DimUnknown
)

type Conversion struct {
	Dimension Dimension
	BaseQty   float64 // 归一到基准单位后的数量；无量纲为 0
	BaseUnit  string  // g / ml / 个 / 适量 / 原始单位
	OrigUnit  string
	Ok        bool // 是否完成了已知换算（unknown 也算 Ok=true 但不换算）
}

var weightToGram = map[string]float64{
	"g":  1,
	"克":  1,
	"kg": 1000,
	"千克": 1000,
	"公斤": 1000,
	"斤":  500,
	"两":  50,
}

var volumeToML = map[string]float64{
	"ml": 1,
	"毫升": 1,
	"l":  1000,
	"L":  1000,
	"升":  1000,
}

var countAlias = map[string]string{
	"个": "个",
	"只": "个",
	"枚": "个",
}

var dimensionless = map[string]bool{
	"适量": true,
	"少许": true,
	"若干": true,
	"to-taste": true,
}

// NormalizeUnit 去掉空白并小写拉丁字母，保留中文。
func NormalizeUnit(u string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range u {
		if unicode.IsSpace(r) {
			continue
		}
		if r <= unicode.MaxASCII {
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func ClassifyUnit(unit string) Dimension {
	u := NormalizeUnit(unit)
	if u == "" {
		return DimUnknown
	}
	if dimensionless[u] {
		return DimDimensionless
	}
	if _, ok := weightToGram[u]; ok {
		return DimWeight
	}
	if _, ok := volumeToML[u]; ok {
		return DimVolume
	}
	if _, ok := countAlias[u]; ok {
		return DimCount
	}
	return DimUnknown
}

// Convert 将数量换算到基准单位。禁止对「把/根/片」等未知单位伪造系数。
func Convert(qty float64, unit string) Conversion {
	u := NormalizeUnit(unit)
	c := Conversion{OrigUnit: u, Ok: true}
	switch ClassifyUnit(u) {
	case DimWeight:
		c.Dimension = DimWeight
		c.BaseUnit = "g"
		c.BaseQty = qty * weightToGram[u]
	case DimVolume:
		c.Dimension = DimVolume
		c.BaseUnit = "ml"
		c.BaseQty = qty * volumeToML[u]
	case DimCount:
		c.Dimension = DimCount
		c.BaseUnit = "个"
		c.BaseQty = qty
	case DimDimensionless:
		c.Dimension = DimDimensionless
		c.BaseUnit = "适量"
		c.BaseQty = 0
	default:
		c.Dimension = DimUnknown
		c.BaseUnit = u
		c.BaseQty = qty
	}
	return c
}

// CanMerge 同量纲可合并；unknown 仅当单位完全相同。
func CanMerge(a, b Conversion) bool {
	if a.Dimension != b.Dimension {
		return false
	}
	if a.Dimension == DimUnknown {
		return a.BaseUnit == b.BaseUnit
	}
	return true
}

func AlmostEqual(a, b float64) bool {
	return math.Abs(a-b) < FloatTol
}

// PrettyQty 把基准量转回更适合阅读的单位（重量超过 500g 用斤）。
func PrettyQty(dim Dimension, baseQty float64, fallbackUnit string) (float64, string) {
	switch dim {
	case DimWeight:
		if baseQty >= 500 && AlmostEqual(math.Mod(baseQty, 500), 0) {
			return baseQty / 500, "斤"
		}
		if baseQty >= 1000 {
			return round3(baseQty / 1000), "kg"
		}
		return round3(baseQty), "g"
	case DimVolume:
		if baseQty >= 1000 {
			return round3(baseQty / 1000), "L"
		}
		return round3(baseQty), "ml"
	case DimCount:
		return round3(baseQty), "个"
	case DimDimensionless:
		return 0, "适量"
	default:
		return round3(baseQty), fallbackUnit
	}
}

func round3(v float64) float64 {
	return math.Round(v*1000) / 1000
}

// Display 生成「鸡蛋 5 个」这类文案。
func Display(name string, dim Dimension, baseQty float64, fallbackUnit string) string {
	q, u := PrettyQty(dim, baseQty, fallbackUnit)
	if dim == DimDimensionless {
		return name + " 适量"
	}
	if AlmostEqual(q, math.Round(q)) {
		return name + " " + formatFloat(math.Round(q)) + " " + u
	}
	return name + " " + formatFloat(q) + " " + u
}
