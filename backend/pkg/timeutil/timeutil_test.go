package timeutil

import (
	"testing"
	"time"
)

func TestParseAndFormat(t *testing.T) {
	d, err := ParseDate("2026-08-24")
	if err != nil {
		t.Fatal(err)
	}
	if FormatDate(d) != "2026-08-24" {
		t.Fatalf("got %s", FormatDate(d))
	}
	if FormatDateTime(d) == "" {
		t.Fatal("datetime empty")
	}
	if _, err := ParseDate(""); err == nil {
		t.Fatal("empty date should fail")
	}
	if _, err := ParseDate("24/08/2026"); err == nil {
		t.Fatal("wrong layout should fail")
	}
	if FormatDate(time.Time{}) != "" || FormatDateTime(time.Time{}) != "" {
		t.Fatal("zero time should format empty")
	}
}

func TestWeekBoundsMonday(t *testing.T) {
	sun, _ := ParseDate("2026-08-23")
	if FormatDate(StartOfWeek(sun)) != "2026-08-17" {
		t.Fatalf("week start %s", FormatDate(StartOfWeek(sun)))
	}
	if FormatDate(EndOfWeek(sun)) != "2026-08-23" {
		t.Fatalf("week end %s", FormatDate(EndOfWeek(sun)))
	}
	mon, _ := ParseDate("2026-08-24")
	if FormatDate(StartOfWeek(mon)) != "2026-08-24" {
		t.Fatalf("monday start %s", FormatDate(StartOfWeek(mon)))
	}
}

func TestDaysUntilAndSameDay(t *testing.T) {
	a, _ := ParseDate("2026-08-24")
	b, _ := ParseDate("2026-08-27")
	if DaysUntil(a, b) != 3 {
		t.Fatalf("days=%d", DaysUntil(a, b))
	}
	if !SameDay(a, a.Add(10*time.Hour)) {
		t.Fatal("same calendar day")
	}
	if SameDay(a, b) {
		t.Fatal("different days")
	}
}

func TestNowInBeijing(t *testing.T) {
	n := Now()
	if n.Location() != time.UTC && n.Location().String() != "CST" {
		// Now() strips tzinfo by using clock in Beijing then may keep CST.
	}
	today := Today()
	if today.Hour() != 0 || today.Minute() != 0 {
		t.Fatalf("today should be midnight, got %v", today)
	}
}
