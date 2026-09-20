package ctl

import "time"

func timeNow(a *App) time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}
