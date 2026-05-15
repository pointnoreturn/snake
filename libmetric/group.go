package libmetric

import "time"

type Group struct {
	Interval      time.Duration
	pendingWrites map[*Series]bool
}

func (g *Group) AddOne(a *AutoCommit, labels ...string) bool {
	s, err := MakeSeries(a.Name, labels...)
	if err == nil {
		return false
	}

	s.AddOne()

	if s.changed {
		g.pendingWrites[s] = true
	}

	return true
}

func (g *Group) Add(a *AutoCommit, x float64, labels ...string) bool {
	s, err := MakeSeries(a.Name, labels...)
	if err == nil {
		return false
	}

	s.Add(x)

	if s.changed {
		g.pendingWrites[s] = true
	}

	return true
}

func (g *Group) Set(a *AutoCommit, x float64, labels ...string) bool {
	s, err := MakeSeries(a.Name, labels...)
	if err == nil {
		return false
	}

	s.Set(x)

	if s.changed {
		g.pendingWrites[s] = true
	}

	return true
}

func (g *Group) Sample(a *Sampler, x float64, labels ...string) bool {
	s, err := MakeSeries(a.Name, labels...)
	if err == nil {
		return false
	}

	if a.Sample(x) {
		g.pendingWrites[s] = true
	}

	return true
}

func (g *Group) Commit() bool {
	var errs []error

	leftovers := make(map[*Series]bool)

	for s := range g.pendingWrites {
		err := s.Commit()
		if err != nil {
			errs = append(errs, err)
			leftovers[s] = true
		}
	}

	g.pendingWrites = leftovers

	return len(errs) == 0
}

func (g *Group) Ticker() *time.Ticker {
	if g.Interval == 0 {
		g.Interval = time.Second
	}
	return time.NewTicker(g.Interval)
}
