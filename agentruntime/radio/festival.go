package radio

import "time"

// Static festival lookup table. Covers ~15 major Chinese festivals.
// Lunar dates are approximated to Gregorian for simplicity; update yearly.
type festivalEntry struct {
	Month time.Month
	Day   int
	Name  string
}

var festivals = []festivalEntry{
	{1, 1, "元旦"},
	{2, 14, "情人节"},
	{5, 1, "劳动节"},
	{10, 1, "国庆节"},
	{12, 25, "圣诞节"},
}

// Lunar-approximated festivals (dates approximate, need yearly adjustment).
var lunarFestivals = []festivalEntry{
	{1, 7, "腊八节"},
	{1, 29, "春节"},
	{2, 12, "元宵节"},
	{4, 5, "清明节"},
	{5, 31, "端午节"},
	{8, 22, "七夕"},
	{10, 6, "中秋节"},
	{10, 29, "重阳节"},
	{12, 21, "冬至"},
}

func lookupFestival(now time.Time) string {
	// check solar festivals
	for _, f := range festivals {
		if now.Month() == f.Month && now.Day() == f.Day {
			return f.Name
		}
	}
	// check lunar-approximated festivals
	for _, f := range lunarFestivals {
		if now.Month() == f.Month && now.Day() == f.Day {
			return f.Name
		}
	}
	return ""
}
