package radio

import (
	"fmt"
	"time"
)

// Context holds external perception info for script generation.
type Context struct {
	Time     string
	Festival string
	State    string
}

func BuildContext(state string) Context {
	now := time.Now()
	wd := now.Weekday().String()
	wdCN := map[string]string{
		"Sunday": "周日", "Monday": "周一", "Tuesday": "周二",
		"Wednesday": "周三", "Thursday": "周四", "Friday": "周五", "Saturday": "周六",
	}[wd]
	hour := now.Hour()

	var period string
	switch {
	case hour < 6:
		period = "凌晨"
	case hour < 9:
		period = "早上"
	case hour < 12:
		period = "上午"
	case hour < 14:
		period = "中午"
	case hour < 18:
		period = "下午"
	default:
		period = "深夜"
	}

	timeStr := fmt.Sprintf("%s%s %s，%s",
		period, wdCN,
		now.Format("15:04"),
		now.Format("2006年1月2日"),
	)

	if state == "" {
		state = defaultState(hour)
	}

	return Context{
		Time:     timeStr,
		Festival: lookupFestival(now),
		State:    state,
	}
}

func defaultState(hour int) string {
	switch {
	case hour < 6:
		return "深夜，听众想听一首歌放松一下"
	case hour < 12:
		return "早上，听众刚起来"
	case hour < 18:
		return "下午，听众在休闲"
	default:
		return "晚上，听众在放松"
	}
}
