package engine

import (
	"testing"

	"gocooking/pkg/timeutil"
)

// TestFullPipeline 覆盖需求六步流水线：展开→归一→聚合→FEFO→过滤→归组准备。
func TestFullPipeline(t *testing.T) {
	lines := []Line{
		{IngredientID: 1, Name: "鸡蛋", Quantity: 2, Unit: "个", MealDate: day("2026-08-24"), Slot: "dinner", RecipeName: "西红柿炒鸡蛋", Multiplier: 1},
		{IngredientID: 1, Name: "鸡蛋", Quantity: 3, Unit: "枚", MealDate: day("2026-08-25"), Slot: "lunch", RecipeName: "蛋炒饭", Multiplier: 1},
		{IngredientID: 9, Name: "盐", Quantity: 1, Unit: "适量", MealDate: day("2026-08-24"), Slot: "dinner", RecipeName: "西红柿炒鸡蛋", Multiplier: 1},
		{IngredientID: 9, Name: "盐", Quantity: 1, Unit: "少许", MealDate: day("2026-08-25"), Slot: "lunch", RecipeName: "蛋炒饭", Multiplier: 1},
		{IngredientID: 88, Name: "花椒", Quantity: 8, Unit: "g", MealDate: day("2026-08-24"), Slot: "dinner", RecipeName: "酸菜鱼", Multiplier: 1},
		{IngredientID: 2, Name: "猪肉", Quantity: 200, Unit: "g", MealDate: day("2026-08-26"), Slot: "dinner", RecipeName: "回锅肉", Multiplier: 1},
		{IngredientID: 2, Name: "猪肉", Quantity: 0.5, Unit: "斤", MealDate: day("2026-08-26"), Slot: "lunch", RecipeName: "青椒肉丝", Multiplier: 1},
	}

	daily := AggregateDaily(lines)
	if len(daily) == 0 {
		t.Fatal("按日聚合不能为空")
	}

	lots := []Lot{
		{ID: 1, IngredientID: 1, Name: "鸡蛋", Quantity: 4, Unit: "个", ExpiresAt: day("2026-08-25")},
		{ID: 2, IngredientID: 2, Name: "猪肉", Quantity: 100, Unit: "g", ExpiresAt: day("2026-08-23")},
	}
	ded := DeductFEFO(daily, lots, day("2026-08-23"))
	merged := MergeAcrossDays(ded.Remaining)

	var eggs, pork, salt, huajiao *MergedItem
	for i := range merged {
		switch merged[i].IngredientID {
		case 1:
			eggs = &merged[i]
		case 2:
			pork = &merged[i]
		case 9:
			salt = &merged[i]
		case 88:
			huajiao = &merged[i]
		}
	}
	if eggs == nil || mathAbs(eggs.BaseQty-1) >= FloatTol {
		t.Fatalf("库存扣 4 个后鸡蛋应剩 1 个: %+v", eggs)
	}
	if pork == nil || mathAbs(pork.BaseQty-450) >= FloatTol {
		t.Fatalf("过期猪肉不得扣减，总需求 450g: %+v", pork)
	}
	if salt == nil || salt.Dimension != DimDimensionless {
		t.Fatalf("盐应为适量: %+v", salt)
	}
	if huajiao == nil {
		t.Fatal("花椒不是常备，聚合后必须存在")
	}

	keep, filtered := FilterStaples(merged, map[uint]bool{9: true})
	if len(filtered) != 1 || filtered[0].IngredientID != 9 {
		t.Fatalf("只应过滤盐: %+v", filtered)
	}
	for _, it := range keep {
		if it.IngredientID == 9 {
			t.Fatal("盐不应出现在采购清单")
		}
		if it.IngredientID == 88 && it.BaseQty < 1 {
			t.Fatal("花椒应进入调料摊")
		}
	}

	keep, filtered = RestoreFiltered(keep, filtered, 9, DimDimensionless, "适量")
	found := false
	for _, it := range keep {
		if it.IngredientID == 9 {
			found = true
		}
	}
	if !found || len(filtered) != 0 {
		t.Fatal("临时加回盐失败")
	}

	if timeutil.FormatDate(day("2026-08-24")) != "2026-08-24" {
		t.Fatal("日期格式")
	}
}

func TestPrettyWeightUsesJin(t *testing.T) {
	q, u := PrettyQty(DimWeight, 1000, "g")
	if u != "kg" && u != "斤" {
		// 1000g → kg；500g → 斤。此处确认不会伪造「把」。
		t.Fatalf("unexpected %v %s", q, u)
	}
	if _, u := PrettyQty(DimUnknown, 1, "把"); u != "把" {
		t.Fatalf("未知单位必须原样返回, got %s", u)
	}
}
