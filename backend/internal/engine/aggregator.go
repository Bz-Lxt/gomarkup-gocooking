// Package engine 实现买菜清单六步流水线（顺序不可调换）：
//
//  1. Expand          将 MealPlan 槽位展开为食材行，数量 × 份数倍数
//  2. Convert/归一    别名在服务层已解析为 IngredientID；此处只做单位换算
//  3. AggregateDaily  按 (用餐日, 食材, 量纲) 累加；unknown 再按单位细分
//  4. DeductFEFO      按用餐日升序扣未过期库存，先到期先用（见 fefo.go）
//  5. MergeAcrossDays 把扣减后的剩余重新合成清单行
//  6. FilterStaples   仅按用户常备开关过滤，不看「调料」分类
//
// 矛盾裁定：
//   - C-01 常备 ≠ 调料分类。花椒、豆瓣酱进入调料摊；盐/生抽进已过滤区。
//   - C-02 保质期相对用餐日，不是「今天」。周一能用、周三过期的批次只扣周一。
//   - C-03 禁止伪造换算。1 把香菜绝不能假设成 30g；同名不同量纲并列并 needs_review。
package engine

import (
	"sort"
	"strings"
	"time"

	"gocooking/pkg/timeutil"
)

// Source 记录清单项溯源。
type Source struct {
	Date       time.Time
	Slot       string
	RecipeName string
	Quantity   float64
	Unit       string
}

// Line 展开后的一条食材需求。
type Line struct {
	IngredientID uint
	Name         string
	Aliases      []string
	Quantity     float64
	Unit         string
	MealDate     time.Time
	Slot         string
	RecipeName   string
	Multiplier   float64
	Optional     bool
}

// DailyNeed 按日聚合后的需求。
type DailyNeed struct {
	IngredientID uint
	Name         string
	Date         time.Time
	Dimension    Dimension
	BaseQty      float64
	BaseUnit     string
	OrigUnit     string
	Sources      []Source
}

// MergedItem 跨日再聚合后的清单行。
type MergedItem struct {
	IngredientID uint
	Name         string
	Dimension    Dimension
	BaseQty      float64
	BaseUnit     string
	OrigUnit     string
	NeedsReview  bool
	Sources      []Source
	Siblings     []MergedItem
}

type AggregateResult struct {
	Daily   []DailyNeed
	Merged  []MergedItem
	Reviews []MergedItem
}

// AliasIndex 将别名/正名映射到规范食材。
type AliasIndex map[string]Canonical

type Canonical struct {
	ID   uint
	Name string
}

func BuildAliasIndex(entries []Canonical, aliases map[uint][]string) AliasIndex {
	idx := make(AliasIndex, len(entries)*3)
	for _, e := range entries {
		idx[normName(e.Name)] = e
		for _, a := range aliases[e.ID] {
			idx[normName(a)] = e
		}
	}
	return idx
}

func (idx AliasIndex) Resolve(name string) (Canonical, bool) {
	c, ok := idx[normName(name)]
	return c, ok
}

func normName(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

// Expand 将份数倍数乘到数量上；可选食材默认纳入（前端可忽略）。
func Expand(in Line) Line {
	m := in.Multiplier
	if m <= 0 {
		m = 1
	}
	out := in
	out.Multiplier = m
	c := Convert(in.Quantity, in.Unit)
	if c.Dimension == DimDimensionless {
		return out
	}
	out.Quantity = in.Quantity * m
	return out
}

// AggregateDaily 步骤③ 的按日版本：同食材同量纲累加。
func AggregateDaily(lines []Line) []DailyNeed {
	type key struct {
		id  uint
		day string
		dim Dimension
		u   string
	}
	type bucket struct {
		need DailyNeed
	}
	m := map[key]*bucket{}
	for _, raw := range lines {
		ln := Expand(raw)
		c := Convert(ln.Quantity, ln.Unit)
		k := key{id: ln.IngredientID, day: timeutil.FormatDate(ln.MealDate), dim: c.Dimension}
		if c.Dimension == DimUnknown {
			k.u = c.BaseUnit
		}
		b, ok := m[k]
		if !ok {
			b = &bucket{need: DailyNeed{
				IngredientID: ln.IngredientID,
				Name:         ln.Name,
				Date:         midnight(ln.MealDate),
				Dimension:    c.Dimension,
				BaseUnit:     c.BaseUnit,
				OrigUnit:     c.OrigUnit,
			}}
			m[k] = b
		}
		if c.Dimension != DimDimensionless {
			b.need.BaseQty += c.BaseQty
		}
		b.need.Sources = append(b.need.Sources, Source{
			Date: midnight(ln.MealDate), Slot: ln.Slot, RecipeName: ln.RecipeName,
			Quantity: ln.Quantity, Unit: NormalizeUnit(ln.Unit),
		})
	}
	out := make([]DailyNeed, 0, len(m))
	for _, b := range m {
		out = append(out, b.need)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].Date.Equal(out[j].Date) {
			return out[i].Date.Before(out[j].Date)
		}
		if out[i].IngredientID != out[j].IngredientID {
			return out[i].IngredientID < out[j].IngredientID
		}
		return out[i].Dimension < out[j].Dimension
	})
	return out
}

// MergeAcrossDays 将扣减后的按日剩余重新聚合成清单行。
func MergeAcrossDays(daily []DailyNeed) []MergedItem {
	type key struct {
		id  uint
		dim Dimension
		u   string
	}
	type bucket struct {
		item MergedItem
	}
	m := map[key]*bucket{}
	for _, d := range daily {
		if d.Dimension != DimDimensionless && d.BaseQty <= FloatTol {
			continue
		}
		k := key{id: d.IngredientID, dim: d.Dimension}
		if d.Dimension == DimUnknown {
			k.u = d.BaseUnit
		}
		b, ok := m[k]
		if !ok {
			b = &bucket{item: MergedItem{
				IngredientID: d.IngredientID,
				Name:         d.Name,
				Dimension:    d.Dimension,
				BaseUnit:     d.BaseUnit,
				OrigUnit:     d.OrigUnit,
			}}
			m[k] = b
		}
		if d.Dimension != DimDimensionless {
			b.item.BaseQty += d.BaseQty
		}
		b.item.Sources = append(b.item.Sources, d.Sources...)
	}

	byIng := map[uint][]MergedItem{}
	for _, b := range m {
		byIng[b.item.IngredientID] = append(byIng[b.item.IngredientID], b.item)
	}
	out := make([]MergedItem, 0, len(m))
	dimensions := make([]MergedItem, 0, len(m))
	for _, grouped := range byIng {
		dimensions = append(dimensions[:0], grouped...)
		if len(dimensions) == 1 {
			out = append(out, dimensions[0])
			continue
		}
		// 同名不同量纲：不强行合并，标记 needs_review，主项带 siblings。
		sort.Slice(dimensions, func(i, j int) bool {
			if dimensions[i].Dimension == dimensions[j].Dimension {
				return dimensions[i].BaseUnit < dimensions[j].BaseUnit
			}
			return dimensions[i].Dimension < dimensions[j].Dimension
		})
		primary := dimensions[0]
		primary.NeedsReview = true
		primary.Siblings = dimensions[1:]
		for i := range primary.Siblings {
			primary.Siblings[i].NeedsReview = true
		}
		out = append(out, primary)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].IngredientID < out[j].IngredientID
	})
	return out
}

func midnight(t time.Time) time.Time {
	t = t.In(timeutil.Beijing)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, timeutil.Beijing)
}

func (it MergedItem) Display() string {
	return Display(it.Name, it.Dimension, it.BaseQty, it.BaseUnit)
}
