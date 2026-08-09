package buildphase

import "time"

type Observer func(name string, duration time.Duration)

func Record(observer Observer, name string, started time.Time) {
	if observer != nil {
		observer(name, time.Since(started))
	}
}
