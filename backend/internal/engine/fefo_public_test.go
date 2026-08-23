package engine_test

import (
	"testing"
	"time"

	"gocooking/internal/engine"
)

func TestDeductFEFODoesNotMutateDemandInput(t *testing.T) {
	mealDate := time.Date(2026, time.August, 25, 0, 0, 0, 0, time.Local)
	demands := []engine.DailyNeed{
		{IngredientID: 2, Name: "土豆", Date: mealDate.AddDate(0, 0, 1), Dimension: engine.DimWeight, BaseQty: 300, BaseUnit: "g"},
		{IngredientID: 1, Name: "西红柿", Date: mealDate, Dimension: engine.DimWeight, BaseQty: 500, BaseUnit: "g"},
	}
	lots := []engine.Lot{
		{ID: 10, IngredientID: 1, Name: "西红柿", Quantity: 200, Unit: "g", ExpiresAt: mealDate.AddDate(0, 0, 2)},
	}

	first := engine.DeductFEFO(demands, lots, mealDate)
	if got := first.Remaining[0].BaseQty; got != 300 {
		t.Fatalf("first calculation remaining quantity = %v, want 300", got)
	}

	second := engine.DeductFEFO(demands, nil, mealDate)
	if got := second.Remaining[0].BaseQty; got != 500 {
		t.Fatalf("reusing the same demands without pantry stock returned %v, want 500", got)
	}
	if demands[0].IngredientID != 2 || demands[0].BaseQty != 300 {
		t.Fatalf("input demands were modified: first item = %+v", demands[0])
	}
}
