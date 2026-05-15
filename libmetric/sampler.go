package libmetric

import (
	"sort"

	"gonum.org/v1/gonum/stat"
)

type SampleF func([]float64) float64

var (
	Average SampleF = func(row []float64) float64 {
		return stat.Mean(row, nil)
	}

	MedianP50 SampleF = func(row []float64) float64 {
		if len(row) == 0 {
			return 0
		}

		cp := append([]float64(nil), row...)
		sort.Float64s(cp)

		return stat.Quantile(0.5, stat.Empirical, cp, nil)
	}

	Median SampleF = func(row []float64) float64 {
		n := len(row)
		if n == 0 {
			return 0
		}

		// copy to avoid mutating input
		cp := make([]float64, n)
		copy(cp, row)

		sort.Float64s(cp)

		mid := n / 2

		if n%2 == 0 {
			return (cp[mid-1] + cp[mid]) / 2
		}

		return cp[mid]
	}

	Sum SampleF = func(row []float64) float64 {
		var sum float64
		for i := range row {
			sum += row[i]
		}
		return sum
	}

	Min SampleF = func(row []float64) float64 {
		if len(row) == 0 {
			return 0
		}

		min := row[0]
		for i := 1; i < len(row); i++ {
			if row[i] < min {
				min = row[i]
			}
		}
		return min
	}

	Max SampleF = func(row []float64) float64 {
		if len(row) == 0 {
			return 0
		}

		max := row[0]
		for i := 1; i < len(row); i++ {
			if row[i] > max {
				max = row[i]
			}
		}
		return max
	}
)

type Sampler struct {
	Name     string
	MinCount int     // minimum number of samples before writing metric
	MaxCount int     // maximum number of latest samples sent to function
	Function SampleF // function to calculate value of the metric

	samples map[string][]float64
}

func (s *Sampler) Sample(x float64, labels ...string) bool {
	c, err := MakeSeries(s.Name, labels...)
	if err != nil {
		// todo log
		return false
	}

	if s.samples == nil {
		s.samples = make(map[string][]float64)
		s.samples[c.key] = []float64{x}
	} else if old := s.samples[c.key]; old != nil {
		s.samples[c.key] = append(old, x)
	} else {
		s.samples[c.key] = []float64{x}
	}

	if s.MaxCount <= 0 {
		s.MaxCount = 100
	}

	if len(s.samples[c.key]) > s.MaxCount {
		s.samples[c.key] = s.samples[c.key][1:]
	}

	if s.MinCount > 0 && len(s.samples[c.key]) < s.MinCount {
		return false
	}

	if s.Function == nil {
		s.Function = Sum
	}

	var value float64 = s.Function(s.samples[c.key])

	c.Set(value)

	return true
}
