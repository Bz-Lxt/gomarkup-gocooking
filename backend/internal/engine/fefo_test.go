package engine

import (
	"testing"

	"gocooking/pkg/timeutil"
)

func TestFEFODeductsUnexpiredOnly(t *testing.T) {
	daily := []DailyNeed{
		{IngredientID: 1, Name: "鸡蛋", Date: day("2026-08-24"), Dimension: DimCount, BaseQty: 3, BaseUnit: "个"},
		{IngredientID: 1, Name: "鸡蛋", Date: day("2026-08-27"), Dimension: DimCount, BaseQty: 3, BaseUnit: "个"},
	}
	lots := []Lot{
		{ID: 11, IngredientID: 1, Name: "鸡蛋", Quantity: 4, Unit: "个", ExpiresAt: day("2026-08-25")},
	}
	res := DeductFEFO(daily, lots, day("2026-08-23"))
	var mon, wed DailyNeed
	for _, n := range res.Remaining {
		if timeutil.FormatDate(n.Date) == "2026-08-24" {
			mon = n
		}
		if timeutil.FormatDate(n.Date) == "2026-08-27" {
			wed = n
		}
	}
	if mathAbs(mon.BaseQty) >= FloatTol {
		t.Fatalf("周一应被库存扣完, leftover=%v", mon.BaseQty)
	}
	if mathAbs(wed.BaseQty-3) >= FloatTol {
		t.Fatalf("周三不得扣过期库存, leftover=%v", wed.BaseQty)
	}
	if len(res.Deducted) == 0 {
		t.Fatal("应有扣减记录")
	}
}

func TestExpiredLotNeverDeducted(t *testing.T) {
	daily := []DailyNeed{
		{IngredientID: 2, Name: "生菜", Date: day("2026-08-24"), Dimension: DimWeight, BaseQty: 200, BaseUnit: "g"},
	}
	lots := []Lot{
		{ID: 1, IngredientID: 2, Name: "生菜", Quantity: 200, Unit: "g", ExpiresAt: day("2026-08-23")},
	}
	res := DeductFEFO(daily, lots, day("2026-08-24"))
	if mathAbs(res.Remaining[0].BaseQty-200) >= FloatTol {
		t.Fatalf("过期库存误扣, leftover=%v", res.Remaining[0].BaseQty)
	}
	if len(res.Deducted) != 0 {
		t.Fatalf("过期批次不得出现在 deducted: %+v", res.Deducted)
	}
}

func TestExpiryEqualsMealDayIsUsable(t *testing.T) {
	daily := []DailyNeed{
		{IngredientID: 3, Name: "豆腐", Date: day("2026-08-24"), Dimension: DimWeight, BaseQty: 300, BaseUnit: "g"},
	}
	lots := []Lot{
		{ID: 2, IngredientID: 3, Name: "豆腐", Quantity: 300, Unit: "g", ExpiresAt: day("2026-08-24")},
	}
	res := DeductFEFO(daily, lots, day("2026-08-23"))
	if mathAbs(res.Remaining[0].BaseQty) >= FloatTol {
		t.Fatalf("保质期=用餐日当天应可用, leftover=%v", res.Remaining[0].BaseQty)
	}
}

func TestFEFOUsesEarliestLotFirst(t *testing.T) {
	daily := []DailyNeed{
		{IngredientID: 4, Name: "牛奶", Date: day("2026-08-24"), Dimension: DimVolume, BaseQty: 200, BaseUnit: "ml"},
	}
	lots := []Lot{
		{ID: 20, IngredientID: 4, Name: "牛奶", Quantity: 500, Unit: "ml", ExpiresAt: day("2026-08-30")},
		{ID: 10, IngredientID: 4, Name: "牛奶", Quantity: 200, Unit: "ml", ExpiresAt: day("2026-08-25")},
	}
	res := DeductFEFO(daily, lots, day("2026-08-23"))
	if len(res.Deducted) != 1 || res.Deducted[0].LotID != 10 {
		t.Fatalf("应先用最早过期批次: %+v", res.Deducted)
	}
}

func TestFilterStaplesOnlyEnabled(t *testing.T) {
	items := []MergedItem{
		{IngredientID: 9, Name: "盐", Dimension: DimDimensionless},
		{IngredientID: 88, Name: "花椒", Dimension: DimWeight, BaseQty: 10, BaseUnit: "g"},
	}
	keep, filtered := FilterStaples(items, map[uint]bool{9: true})
	if len(keep) != 1 || keep[0].Name != "花椒" {
		t.Fatalf("花椒不应被过滤: %+v", keep)
	}
	if len(filtered) != 1 || filtered[0].Name != "盐" {
		t.Fatalf("盐应被过滤: %+v", filtered)
	}
}

func TestRestoreFiltered(t *testing.T) {
	keep := []MergedItem{}
	filtered := []MergedItem{{IngredientID: 9, Name: "盐", Dimension: DimDimensionless, BaseUnit: "适量"}}
	keep, filtered = RestoreFiltered(keep, filtered, 9, DimDimensionless, "适量")
	if len(keep) != 1 || len(filtered) != 0 {
		t.Fatalf("加回失败 keep=%+v filtered=%+v", keep, filtered)
	}
}

func TestPartialLotLeavesRemainder(t *testing.T) {
	daily := []DailyNeed{
		{IngredientID: 1, Name: "鸡蛋", Date: day("2026-08-24"), Dimension: DimCount, BaseQty: 5, BaseUnit: "个"},
	}
	lots := []Lot{
		{ID: 1, IngredientID: 1, Name: "鸡蛋", Quantity: 2, Unit: "个", ExpiresAt: day("2026-08-28")},
	}
	res := DeductFEFO(daily, lots, day("2026-08-23"))
	if mathAbs(res.Remaining[0].BaseQty-3) >= FloatTol {
		t.Fatalf("扣 2 剩 3, got %v", res.Remaining[0].BaseQty)
	}
	if mathAbs(res.Deducted[0].FromPantry-2) >= FloatTol {
		t.Fatalf("deducted=%v", res.Deducted[0].FromPantry)
	}
}

func TestUnknownUnitOnlyMatchesSameUnit(t *testing.T) {
	daily := []DailyNeed{
		{IngredientID: 7, Name: "香菜", Date: day("2026-08-24"), Dimension: DimUnknown, BaseQty: 1, BaseUnit: "把"},
	}
	lots := []Lot{
		{ID: 1, IngredientID: 7, Name: "香菜", Quantity: 20, Unit: "g", ExpiresAt: day("2026-08-28")},
	}
	res := DeductFEFO(daily, lots, day("2026-08-23"))
	if mathAbs(res.Remaining[0].BaseQty-1) >= FloatTol {
		t.Fatal("g 库存不得扣减 把 需求")
	}
}

func TestVolumeFEFO(t *testing.T) {
	daily := []DailyNeed{
		{IngredientID: 8, Name: "牛奶", Date: day("2026-08-24"), Dimension: DimVolume, BaseQty: 300, BaseUnit: "ml"},
	}
	lots := []Lot{
		{ID: 1, IngredientID: 8, Name: "牛奶", Quantity: 0.2, Unit: "L", ExpiresAt: day("2026-08-25")},
		{ID: 2, IngredientID: 8, Name: "牛奶", Quantity: 200, Unit: "ml", ExpiresAt: day("2026-08-29")},
	}
	res := DeductFEFO(daily, lots, day("2026-08-23"))
	if mathAbs(res.Remaining[0].BaseQty) >= FloatTol {
		t.Fatalf("0.2L + 200ml 应扣完 300ml, leftover=%v", res.Remaining[0].BaseQty)
	}
	if len(res.Deducted) < 2 {
		t.Fatalf("应跨两批次扣减: %+v", res.Deducted)
	}
}

func TestAlertForSoonAndExpired(t *testing.T) {
	daily := []DailyNeed{}
	lots := []Lot{
		{ID: 1, IngredientID: 2, Name: "生菜", Quantity: 100, Unit: "g", ExpiresAt: day("2026-08-22")},
		{ID: 2, IngredientID: 3, Name: "豆腐", Quantity: 100, Unit: "g", ExpiresAt: day("2026-08-25")},
	}
	res := DeductFEFO(daily, lots, day("2026-08-23"))
	if len(res.Alerts) < 2 {
		t.Fatalf("应同时提示过期与临期: %+v", res.Alerts)
	}
}

func TestNoRedundantPurchase(t *testing.T) {
	daily := []DailyNeed{
		{IngredientID: 1, Name: "鸡蛋", Date: day("2026-08-24"), Dimension: DimCount, BaseQty: 5, BaseUnit: "个"},
	}
	lots := []Lot{
		{ID: 1, IngredientID: 1, Name: "鸡蛋", Quantity: 5, Unit: "个", ExpiresAt: day("2026-08-28")},
	}
	res := DeductFEFO(daily, lots, day("2026-08-23"))
	merged := MergeAcrossDays(res.Remaining)
	if len(merged) != 0 {
		t.Fatalf("库存充足时清单不得出现该食材: %+v", merged)
	}
}
