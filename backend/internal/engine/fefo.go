// FEFO（First Expire First Out）扣减规则：
//
//   - 批次按 ExpiresAt 升序、再按 ID 升序
//   - 仅当 date(ExpiresAt) >= date(MealDate) 才允许扣减
//   - 保质期 = 用餐日当天：可用
//   - 保质期 = 用餐日前一天：不可用（过期库存误扣率必须为 0）
//   - 量纲必须一致；unknown 还要求单位字符串完全相同
//   - 无量纲（适量）不参与数量扣减
//
// 扣减完成后，仍有剩余且 3 天内到期或已过期的批次会进入 ExpiryAlert，
// 供清单页提示「建议提前食用」。
package engine

import (
	"sort"
	"time"

	"gocooking/pkg/timeutil"
)

// Lot 冰箱中的一笔库存。
type Lot struct {
	ID           uint
	IngredientID uint
	Name         string
	Quantity     float64
	Unit         string
	ExpiresAt    time.Time
}

// DeductRecord 记录从库存扣掉了多少。
type DeductRecord struct {
	IngredientID uint
	Name         string
	FromPantry   float64
	Unit         string
	Dimension    Dimension
	LotID        uint
	MealDate     time.Time
}

type ExpiryAlert struct {
	IngredientID uint
	Name         string
	ExpiresAt    time.Time
	Message      string
}

type DeductResult struct {
	Remaining []DailyNeed
	Deducted  []DeductRecord
	Alerts    []ExpiryAlert
}

type lotState struct {
	Lot
	Conv      Conversion
	Remaining float64
}

// DeductFEFO 按用餐日升序、先到期先用。保质期 < 用餐日 的批次不得扣减。
func DeductFEFO(daily []DailyNeed, lots []Lot, today time.Time) DeductResult {
	states := make([]*lotState, 0, len(lots))
	for _, l := range lots {
		c := Convert(l.Quantity, l.Unit)
		states = append(states, &lotState{Lot: l, Conv: c, Remaining: c.BaseQty})
	}
	sort.Slice(states, func(i, j int) bool {
		if !states[i].ExpiresAt.Equal(states[j].ExpiresAt) {
			return states[i].ExpiresAt.Before(states[j].ExpiresAt)
		}
		return states[i].ID < states[j].ID
	})

	needs := daily
	sort.Slice(needs, func(i, j int) bool {
		if !needs[i].Date.Equal(needs[j].Date) {
			return needs[i].Date.Before(needs[j].Date)
		}
		return needs[i].IngredientID < needs[j].IngredientID
	})

	var deducted []DeductRecord
	for i := range needs {
		n := &needs[i]
		if n.Dimension == DimDimensionless {
			continue
		}
		needLeft := n.BaseQty
		for _, st := range states {
			if needLeft <= FloatTol {
				break
			}
			if st.IngredientID != n.IngredientID {
				continue
			}
			if st.Conv.Dimension != n.Dimension {
				continue
			}
			if st.Conv.Dimension == DimUnknown && st.Conv.BaseUnit != n.BaseUnit {
				continue
			}
			if dateOnly(st.ExpiresAt).Before(dateOnly(n.Date)) {
				continue
			}
			if st.Remaining <= FloatTol {
				continue
			}
			take := st.Remaining
			if take > needLeft {
				take = needLeft
			}
			st.Remaining -= take
			needLeft -= take
			deducted = append(deducted, DeductRecord{
				IngredientID: n.IngredientID,
				Name:         n.Name,
				FromPantry:   take,
				Unit:         n.BaseUnit,
				Dimension:    n.Dimension,
				LotID:        st.ID,
				MealDate:     n.Date,
			})
		}
		n.BaseQty = needLeft
		if n.BaseQty < 0 {
			n.BaseQty = 0
		}
	}

	alerts := buildAlerts(states, today)
	return DeductResult{Remaining: needs, Deducted: deducted, Alerts: alerts}
}

func buildAlerts(states []*lotState, today time.Time) []ExpiryAlert {
	today = dateOnly(today)
	seen := map[uint]ExpiryAlert{}
	for _, st := range states {
		if st.Remaining <= FloatTol {
			continue
		}
		exp := dateOnly(st.ExpiresAt)
		days := timeutil.DaysUntil(today, exp)
		if days < 0 {
			seen[st.IngredientID] = ExpiryAlert{
				IngredientID: st.IngredientID,
				Name:         st.Name,
				ExpiresAt:    exp,
				Message:      "冰箱中" + st.Name + "已于 " + timeutil.FormatDate(exp) + " 过期，未被扣减",
			}
			continue
		}
		if days <= 3 {
			if old, ok := seen[st.IngredientID]; ok && !old.ExpiresAt.After(exp) {
				continue
			}
			seen[st.IngredientID] = ExpiryAlert{
				IngredientID: st.IngredientID,
				Name:         st.Name,
				ExpiresAt:    exp,
				Message:      "冰箱中" + st.Name + "将于 " + timeutil.FormatDate(exp) + " 过期，建议提前食用",
			}
		}
	}
	out := make([]ExpiryAlert, 0, len(seen))
	for _, a := range seen {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ExpiresAt.Before(out[j].ExpiresAt) })
	return out
}

func dateOnly(t time.Time) time.Time {
	t = t.In(timeutil.Beijing)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, timeutil.Beijing)
}

// FilterStaples 仅过滤 enabled=true 的常备项，调料分类本身不作为过滤依据。
func FilterStaples(items []MergedItem, stapleOn map[uint]bool) (keep, filtered []MergedItem) {
	for _, it := range items {
		if stapleOn[it.IngredientID] {
			filtered = append(filtered, it)
			continue
		}
		keep = append(keep, it)
	}
	return keep, filtered
}

// RestoreFiltered 把指定食材从已过滤列表加回清单（盐用完了）。
func RestoreFiltered(keep, filtered []MergedItem, id uint, dim Dimension, unit string) (nextKeep, nextFiltered []MergedItem) {
	unit = NormalizeUnit(unit)
	for _, it := range filtered {
		if it.IngredientID == id && it.Dimension == dim && (dim != DimUnknown || NormalizeUnit(it.BaseUnit) == unit) {
			keep = append(keep, it)
			continue
		}
		nextFiltered = append(nextFiltered, it)
	}
	return keep, nextFiltered
}
