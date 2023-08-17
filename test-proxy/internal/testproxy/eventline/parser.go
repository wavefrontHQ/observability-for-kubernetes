package eventline

type Event struct {
	Name        string
	Start       string
	End         string
	Annotations map[string]string
	Tags        map[string]string
}

func Parse(line string) (*Event, error) {
	g := &EventGrammar{Buffer: line}
	g.Init()
	if err := g.Parse(); err != nil {
		return nil, err
	}
	g.Execute()
	return &Event{
		Name:        g.Name,
		Start:       g.StartMillis,
		End:         g.EndMillis,
		Annotations: g.Annotations,
		Tags:        g.Tags,
	}, nil
}
