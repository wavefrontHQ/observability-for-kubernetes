package metricline

type Metric struct {
	Name      string
	Value     string
	Timestamp string
	Tags      map[string]string
	Buckets   map[string]string
}

func Parse(line string) (*Metric, error) {
	g := &MetricGrammar{Buffer: line}
	g.Init()
	if err := g.Parse(); err != nil {
		return nil, err
	}
	g.Execute()
	m := &Metric{
		Name:      g.Name,
		Timestamp: g.Timestamp,
		Tags:      g.Tags,
	}
	if g.Histogram {
		m.Buckets = g.Buckets
	} else {
		m.Value = g.Value
	}
	return m, nil
}
