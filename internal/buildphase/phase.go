package buildphase

import "time"

type Observer func(name string, duration time.Duration)

func Record(observer Observer, name string, started time.Time) {
	Emit(observer, name, time.Since(started))
}

func Emit(observer Observer, name string, duration time.Duration) {
	if observer != nil {
		observer(name, duration)
	}
}
