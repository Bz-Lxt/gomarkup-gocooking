// Package timeutil 封装北京时间。禁止业务侧 time.Now().UTC() 直接落库。
package timeutil

import (
	"fmt"
	"time"
)

// Beijing 为全项目唯一允许的业务时区（GMT+8）。
var Beijing = time.FixedZone("CST", 8*60*60)

const (
	DateLayout     = "2006-01-02"
	DateTimeLayout = "2006-01-02 15:04:05"
)

// Now 返回不带时区信息的北京时间，供 GORM 落库使用。
func Now() time.Time {
	return time.Now().In(Beijing).Truncate(time.Second)
}

// Today 返回北京时间当天 00:00:00。
func Today() time.Time {
	n := time.Now().In(Beijing)
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, Beijing)
}

// ParseDate 解析 yyyy-MM-dd，结果落在北京时区零点。
func ParseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}
	t, err := time.ParseInLocation(DateLayout, s, Beijing)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q: %w", s, err)
	}
	return t, nil
}

// FormatDate 输出 yyyy-MM-dd。
func FormatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(Beijing).Format(DateLayout)
}

// FormatDateTime 输出用户可见的北京时间。
func FormatDateTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(Beijing).Format(DateTimeLayout)
}

// StartOfWeek 返回包含 d 的周一 00:00（北京）。
func StartOfWeek(d time.Time) time.Time {
	d = midnight(d)
	wd := int(d.Weekday())
	if wd == 0 {
		wd = 7
	}
	return d.AddDate(0, 0, 1-wd)
}

// EndOfWeek 返回该周日 00:00（北京）。
func EndOfWeek(d time.Time) time.Time {
	return StartOfWeek(d).AddDate(0, 0, 6)
}

// SameDay 比较两个时间是否为同一北京日历日。
func SameDay(a, b time.Time) bool {
	return FormatDate(a) == FormatDate(b)
}

// DaysUntil 返回 from 到 to 的日历天数（to - from）。
func DaysUntil(from, to time.Time) int {
	a := midnight(from)
	b := midnight(to)
	return int(b.Sub(a).Hours() / 24)
}

func midnight(t time.Time) time.Time {
	t = t.In(Beijing)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, Beijing)
}
