package engine

import (
	"math"
	"testing"
)

func TestConvertWeight(t *testing.T) {
	cases := []struct {
		qty  float64
		unit string
		want float64
	}{
		{200, "g", 200},
		{200, "克", 200},
		{0.5, "斤", 250},
		{1, "kg", 1000},
		{1, "千克", 1000},
		{2, "两", 100},
		{1, "公斤", 1000},
	}
	for _, tc := range cases {
		got := Convert(tc.qty, tc.unit)
		if got.Dimension != DimWeight {
			t.Fatalf("%v %s dim=%s", tc.qty, tc.unit, got.Dimension)
		}
		if math.Abs(got.BaseQty-tc.want) >= FloatTol {
			t.Fatalf("%v %s => %v want %v (err=%g)", tc.qty, tc.unit, got.BaseQty, tc.want, math.Abs(got.BaseQty-tc.want))
		}
	}
}

func TestConvertVolumeAndCount(t *testing.T) {
	ml := Convert(1.5, "L")
	if ml.Dimension != DimVolume || math.Abs(ml.BaseQty-1500) >= FloatTol {
		t.Fatalf("1.5L => %+v", ml)
	}
	cnt := Convert(3, "枚")
	if cnt.Dimension != DimCount || cnt.BaseUnit != "个" || math.Abs(cnt.BaseQty-3) >= FloatTol {
		t.Fatalf("3枚 => %+v", cnt)
	}
}

func TestDimensionlessAndUnknown(t *testing.T) {
	d := Convert(1, "适量")
	if d.Dimension != DimDimensionless || d.BaseUnit != "适量" {
		t.Fatalf("适量 => %+v", d)
	}
	u := Convert(1, "把")
	if u.Dimension != DimUnknown || u.BaseUnit != "把" || math.Abs(u.BaseQty-1) >= FloatTol {
		t.Fatalf("1把不得伪造换算: %+v", u)
	}
}

func TestCanMerge(t *testing.T) {
	a := Convert(200, "g")
	b := Convert(0.5, "斤")
	if !CanMerge(a, b) {
		t.Fatal("g 与 斤 应可合并")
	}
	c := Convert(1, "把")
	d := Convert(20, "g")
	if CanMerge(c, d) {
		t.Fatal("把 与 g 不得合并")
	}
	e := Convert(1, "把")
	f := Convert(2, "把")
	if !CanMerge(e, f) {
		t.Fatal("同单位 把 应可合并")
	}
}

func TestConvertTableDrivenAllUnits(t *testing.T) {
	cases := []struct {
		qty  float64
		unit string
		dim  Dimension
		base float64
	}{
		{1, "kg", DimWeight, 1000},
		{2, "公斤", DimWeight, 2000},
		{3, "两", DimWeight, 150},
		{1, "斤", DimWeight, 500},
		{250, "毫升", DimVolume, 250},
		{2, "升", DimVolume, 2000},
		{1, "l", DimVolume, 1000},
		{4, "只", DimCount, 4},
		{2, "个", DimCount, 2},
		{1, "若干", DimDimensionless, 0},
		{1, "少许", DimDimensionless, 0},
		{3, "片", DimUnknown, 3},
		{2, "根", DimUnknown, 2},
		{1, "碗", DimUnknown, 1},
	}
	for _, tc := range cases {
		got := Convert(tc.qty, tc.unit)
		if got.Dimension != tc.dim {
			t.Fatalf("%v%s dim=%s want %s", tc.qty, tc.unit, got.Dimension, tc.dim)
		}
		if math.Abs(got.BaseQty-tc.base) >= FloatTol {
			t.Fatalf("%v%s base=%v want %v", tc.qty, tc.unit, got.BaseQty, tc.base)
		}
	}
}

func TestNormalizeUnitStripsSpace(t *testing.T) {
	if NormalizeUnit("  斤 ") != "斤" {
		t.Fatal(NormalizeUnit("  斤 "))
	}
	if ClassifyUnit("KG") != DimWeight {
		t.Fatal("KG should be weight")
	}
}

func TestPrettyAndDisplay(t *testing.T) {
	if got := Display("猪肉", DimWeight, 500, "g"); got != "猪肉 1 斤" {
		t.Fatalf("got %q", got)
	}
	if got := Display("盐", DimDimensionless, 0, "适量"); got != "盐 适量" {
		t.Fatalf("got %q", got)
	}
	if got := Display("鸡蛋", DimCount, 5, "个"); got != "鸡蛋 5 个" {
		t.Fatalf("got %q", got)
	}
}
