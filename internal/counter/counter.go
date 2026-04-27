package counter

import "time"

type Snapshot struct {
	Value     int32
	UpdatedAt time.Time
	Error     string
	Source    string
}

type Change struct {
	Delta int
}
