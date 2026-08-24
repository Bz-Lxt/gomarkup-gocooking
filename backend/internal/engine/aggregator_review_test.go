package engine_test

import (
	"testing"
	"time"

	"gocooking/internal/engine"
)

func TestMergeAcrossDaysKeepsReviewSiblingsWithTheirIngredient(t *testing.T) {
	day := time.Date(2026, time.August, 24, 0, 0, 0, 0, time.UTC)
	daily := []engine.DailyNeed{
		{IngredientID: 3, Name: "香菜", Date: day, Dimension: engine.DimUnknown, BaseQty: 1, BaseUnit: "把", OrigUnit: "把"},
		{IngredientID: 3, Name: "香菜", Date: day, Dimension: engine.DimWeight, BaseQty: 20, BaseUnit: "g", OrigUnit: "g"},
		{IngredientID: 4, Name: "大蒜", Date: day, Dimension: engine.DimUnknown, BaseQty: 2, BaseUnit: "瓣", OrigUnit: "瓣"},
		{IngredientID: 4, Name: "大蒜", Date: day, Dimension: engine.DimWeight, BaseQty: 50, BaseUnit: "g", OrigUnit: "g"},
	}

	got := engine.MergeAcrossDays(daily)
	if len(got) != 2 {
		t.Fatalf("want two review items, got %+v", got)
	}
	for _, item := range got {
		if !item.NeedsReview || len(item.Siblings) != 1 {
			t.Fatalf("ingredient %d should have one review sibling: %+v", item.IngredientID, item)
		}
		sibling := item.Siblings[0]
		if sibling.IngredientID != item.IngredientID || sibling.Name != item.Name {
			t.Fatalf("review sibling crossed ingredients: item=%+v sibling=%+v", item, sibling)
		}
	}
}
