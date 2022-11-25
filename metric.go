package graphigo

import (
	"fmt"
	"time"
)

// Metric represents a single metric to be sent to graphite.
type Metric struct {
	Path      string    // namespace of the metric
	Value     any       // value of the metric, this is formatted using %v
	Timestamp time.Time // time the metric was recorded
}

func (m Metric) String() string {
	return fmt.Sprintf("%s %v %s", m.Path, m.Value, m.Timestamp.UTC().Format(time.RFC3339))
}
