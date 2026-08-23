package engine

import (
	"testing"
	"time"

	"gocooking/pkg/timeutil"
)

func day(s string) time.Time {
	t, err := timeutil.ParseDate(s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestEggsMergeTwoPlusThree(t *testing.T) {
	lines := []Line{
		{IngredientID: 1, Name: "鸡蛋", Quantity: 2, Unit: "个", MealDate: day("2026-08-24"), Slot: "dinner", RecipeName: "西红柿炒鸡蛋", Multiplier: 1},
		{IngredientID: 1, Name: "鸡蛋", Quantity: 3, Unit: "枚", MealDate: day("2026-08-25"), Slot: "lunch", RecipeName: "蛋炒饭", Multiplier: 1},
	}
	daily := AggregateDaily(lines)
	merged := MergeAcrossDays(daily)
	if len(merged) != 1 {
		t.Fatalf("want 1 item got %d", len(merged))
	}
	if mathAbs(merged[0].BaseQty-5) >= FloatTol {
		t.Fatalf("鸡蛋应合并为 5 个, got %v", merged[0].BaseQty)
	}
	if merged[0].Display() != "鸡蛋 5 个" {
		t.Fatalf("display=%q", merged[0].Display())
	}
}

func TestAliasIndex(t *testing.T) {
	idx := BuildAliasIndex(
		[]Canonical{{ID: 10, Name: "西红柿"}},
		map[uint][]string{10: {"番茄", "洋柿子"}},
	)
	for _, n := range []string{"西红柿", "番茄", "洋柿子"} {
		c, ok := idx.Resolve(n)
		if !ok || c.ID != 10 {
			t.Fatalf("%s 未归一到西红柿", n)
		}
	}
}

func TestPorkWeightMerge(t *testing.T) {
	lines := []Line{
		{IngredientID: 2, Name: "猪肉", Quantity: 200, Unit: "g", MealDate: day("2026-08-24"), Slot: "dinner", RecipeName: "青椒肉丝", Multiplier: 1},
		{IngredientID: 2, Name: "猪肉", Quantity: 0.5, Unit: "斤", MealDate: day("2026-08-25"), Slot: "lunch", RecipeName: "回锅肉", Multiplier: 1},
	}
	merged := MergeAcrossDays(AggregateDaily(lines))
	if len(merged) != 1 || mathAbs(merged[0].BaseQty-450) >= FloatTol {
		t.Fatalf("猪肉应合并为 450g, got %+v", merged)
	}
}

func TestCorianderNotFakeMerge(t *testing.T) {
	lines := []Line{
		{IngredientID: 3, Name: "香菜", Quantity: 1, Unit: "把", MealDate: day("2026-08-24"), Slot: "dinner", RecipeName: "酸菜鱼", Multiplier: 1},
		{IngredientID: 3, Name: "香菜", Quantity: 20, Unit: "g", MealDate: day("2026-08-25"), Slot: "lunch", RecipeName: "凉拌黄瓜", Multiplier: 1},
	}
	merged := MergeAcrossDays(AggregateDaily(lines))
	if len(merged) != 1 {
		t.Fatalf("应作为同一食材的主项+siblings, got %d", len(merged))
	}
	if !merged[0].NeedsReview {
		t.Fatal("不同量纲必须 needs_review")
	}
	if len(merged[0].Siblings) != 1 {
		t.Fatalf("siblings=%d", len(merged[0].Siblings))
	}
}

func TestDimensionlessMerge(t *testing.T) {
	lines := []Line{
		{IngredientID: 4, Name: "盐", Quantity: 1, Unit: "适量", MealDate: day("2026-08-24"), Slot: "dinner", RecipeName: "A", Multiplier: 1},
		{IngredientID: 4, Name: "盐", Quantity: 1, Unit: "少许", MealDate: day("2026-08-25"), Slot: "lunch", RecipeName: "B", Multiplier: 2},
	}
	merged := MergeAcrossDays(AggregateDaily(lines))
	if len(merged) != 1 || merged[0].Dimension != DimDimensionless {
		t.Fatalf("%+v", merged)
	}
	if merged[0].Display() != "盐 适量" {
		t.Fatalf("display=%q", merged[0].Display())
	}
}

func TestMultiplierScalesQty(t *testing.T) {
	lines := []Line{
		{IngredientID: 1, Name: "鸡蛋", Quantity: 2, Unit: "个", MealDate: day("2026-08-24"), Slot: "dinner", RecipeName: "西红柿炒鸡蛋", Multiplier: 2},
	}
	merged := MergeAcrossDays(AggregateDaily(lines))
	if mathAbs(merged[0].BaseQty-4) >= FloatTol {
		t.Fatalf("2x 倍数应为 4 个, got %v", merged[0].BaseQty)
	}
}

func TestSixDishMonToWedMerge(t *testing.T) {
	// 需求原始场景：周一到周三安排 6 道菜，鸡蛋 2+3 必须合并。
	lines := []Line{
		{IngredientID: 1, Name: "鸡蛋", Quantity: 2, Unit: "个", MealDate: day("2026-08-24"), Slot: "dinner", RecipeName: "西红柿炒鸡蛋", Multiplier: 1},
		{IngredientID: 10, Name: "西红柿", Quantity: 2, Unit: "个", MealDate: day("2026-08-24"), Slot: "dinner", RecipeName: "西红柿炒鸡蛋", Multiplier: 1},
		{IngredientID: 1, Name: "鸡蛋", Quantity: 2, Unit: "个", MealDate: day("2026-08-25"), Slot: "lunch", RecipeName: "蛋炒饭", Multiplier: 1},
		{IngredientID: 2, Name: "猪肉", Quantity: 200, Unit: "g", MealDate: day("2026-08-25"), Slot: "dinner", RecipeName: "青椒肉丝", Multiplier: 1},
		{IngredientID: 1, Name: "鸡蛋", Quantity: 1, Unit: "枚", MealDate: day("2026-08-26"), Slot: "lunch", RecipeName: "木须肉", Multiplier: 1},
		{IngredientID: 5, Name: "鲈鱼", Quantity: 1, Unit: "条", MealDate: day("2026-08-26"), Slot: "dinner", RecipeName: "清蒸鲈鱼", Multiplier: 1},
	}
	merged := MergeAcrossDays(AggregateDaily(lines))
	var eggs *MergedItem
	for i := range merged {
		if merged[i].IngredientID == 1 {
			eggs = &merged[i]
		}
	}
	if eggs == nil || mathAbs(eggs.BaseQty-5) >= FloatTol {
		t.Fatalf("6 道菜中鸡蛋必须合并为 5, got %+v", merged)
	}
}

func TestZeroMultiplierDefaultsToOne(t *testing.T) {
	ln := Expand(Line{IngredientID: 1, Name: "鸡蛋", Quantity: 2, Unit: "个", Multiplier: 0})
	if mathAbs(ln.Quantity-2) >= FloatTol {
		t.Fatalf("0 倍数应回落为 1: %v", ln.Quantity)
	}
}

func TestHalfMultiplier(t *testing.T) {
	lines := []Line{{IngredientID: 2, Name: "猪肉", Quantity: 200, Unit: "g", MealDate: day("2026-08-24"), Slot: "dinner", RecipeName: "青椒肉丝", Multiplier: 0.5}}
	merged := MergeAcrossDays(AggregateDaily(lines))
	if mathAbs(merged[0].BaseQty-100) >= FloatTol {
		t.Fatalf("0.5x 应为 100g, got %v", merged[0].BaseQty)
	}
}

func TestSameUnknownUnitMerges(t *testing.T) {
	lines := []Line{
		{IngredientID: 3, Name: "香菜", Quantity: 1, Unit: "把", MealDate: day("2026-08-24"), Slot: "dinner", RecipeName: "A", Multiplier: 1},
		{IngredientID: 3, Name: "香菜", Quantity: 2, Unit: "把", MealDate: day("2026-08-25"), Slot: "lunch", RecipeName: "B", Multiplier: 1},
	}
	merged := MergeAcrossDays(AggregateDaily(lines))
	if len(merged) != 1 || merged[0].NeedsReview || mathAbs(merged[0].BaseQty-3) >= FloatTol {
		t.Fatalf("同单位 把 应合并为 3: %+v", merged)
	}
}

func TestEmptyLines(t *testing.T) {
	if got := MergeAcrossDays(AggregateDaily(nil)); len(got) != 0 {
		t.Fatalf("%+v", got)
	}
}

func TestSourcesKeepTrace(t *testing.T) {
	lines := []Line{
		{IngredientID: 1, Name: "鸡蛋", Quantity: 2, Unit: "个", MealDate: day("2026-08-24"), Slot: "dinner", RecipeName: "西红柿炒鸡蛋", Multiplier: 1},
		{IngredientID: 1, Name: "鸡蛋", Quantity: 3, Unit: "个", MealDate: day("2026-08-26"), Slot: "lunch", RecipeName: "蛋炒饭", Multiplier: 1},
	}
	merged := MergeAcrossDays(AggregateDaily(lines))
	if len(merged[0].Sources) != 2 {
		t.Fatalf("溯源条数=%d", len(merged[0].Sources))
	}
}

func mathAbs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
