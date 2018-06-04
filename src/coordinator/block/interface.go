// Copyright (c) 2018 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package block

import (
	"fmt"
	"time"

	"github.com/m3db/m3db/src/coordinator/models"
)

const (
	// ErrBounds is returned when time requested is outside the block bounds
	ErrBounds = "out of bounds, time: %v, bounds: %v"
)

// Block represents a group of series across a time bound
type Block interface {
	Meta() Metadata
	StepIter() StepIter
	SeriesIter() SeriesIter
	SeriesMeta() []SeriesMeta
	Close() error
}

// SeriesMeta is metadata data for the series
type SeriesMeta struct {
	Tags models.Tags
}

// Bounds are the time bounds
// nolint: structcheck, megacheck, unused
type Bounds struct {
	Start    time.Time
	End      time.Time
	StepSize time.Duration
}

// SeriesIter iterates through a CompressedSeriesIterator horizontally
type SeriesIter interface {
	Next() bool
	Current() Series
}

// StepIter iterates through a CompressedStepIterator vertically
type StepIter interface {
	Next() bool
	Current() Step
	Len() int
}

// Step can optionally implement iterator interface
type Step interface {
	Time() time.Time
	Values() []float64
}

// Metadata is metadata for a block
type Metadata struct {
	Bounds Bounds
	Tags   models.Tags // Common tags across different series
}

// Builder builds a new block
type Builder interface {
	AppendValue(index int, value float64) error
	Build() Block
	AddCols(num int) error
}

// Result is the result from a block query
type Result struct {
	Blocks []Block
}

// Series is a single series within a block
type Series struct {
	values []float64
	bounds Bounds
}

// NewSeries creates a new series
func NewSeries(values []float64, bounds Bounds) Series {
	return Series{values: values, bounds: bounds}
}

// ValueAtStep returns the datapoint value at a step index
func (s Series) ValueAtStep(idx int) float64 {
	return s.values[idx]
}

// ValueAtTime returns the datapoint value at a given time
func (s Series) ValueAtTime(t time.Time) (float64, error) {
	if t.Before(s.bounds.Start) || t.After(s.bounds.End) {
		return 0, fmt.Errorf(ErrBounds, t, s.bounds)
	}

	step := int(t.Sub(s.bounds.Start) / s.bounds.StepSize)
	if step >= len(s.values) {
		return 0, fmt.Errorf(ErrBounds, t, s.bounds)
	}

	return s.ValueAtStep(step), nil
}
