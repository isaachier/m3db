package storage

import (
	"fmt"
	"time"

	"github.com/m3db/m3db/src/coordinator/models"
	"github.com/m3db/m3db/src/coordinator/ts"
)

// Block represents a group of series across a time bound
type Block interface {
	Meta() BlockMetadata
	StepIter() StepIter
	SeriesIter() SeriesIter
	SeriesMeta() []SeriesMeta
	StepMeta() []StepMeta
}

// SeriesMeta is metadata data for the series
type SeriesMeta struct {
	Tags models.Tags
}

// StepMeta is metadata data for a single time step
type StepMeta struct {
}

// Bounds are the time bounds
// nolint: structcheck, megacheck, unused
type Bounds struct {
	start    time.Time
	end      time.Time
	stepSize time.Duration
}

// SeriesIter iterates through a CompressedSeriesIterator horizontally
type SeriesIter interface {
	Next() bool
	Current() ts.Series
}

// StepIter iterates through a CompressedStepIterator vertically
type StepIter interface {
	Next() bool
	Current() Step
}

// Step can optionally implement iterator interface
type Step interface {
	Time() time.Time
	Values() []float64
}

// BlockMetadata is metadata for a block
type BlockMetadata struct {
	Bounds Bounds
	Tags   models.Tags // Common tags across different series
}

type ColumnBlockBuilder struct {
	block ColumnBlock
}

type ColumnBlock struct {
	columns [] Column
}

func (c ColumnBlock)

func NewColumnBlockBuilder() ColumnBlockBuilder {
	return ColumnBlockBuilder{}
}

func (cb ColumnBlockBuilder) AppendValue(index int, value float64) error {
	if len(cb.block.columns) <= index {
		return fmt.Errorf("index out of range for append: %d", index)
	}

	cb.block.columns[index].Values = append(cb.block.columns[index].Values, value)
	return nil
}

// TODO: Return an immutable copy
func (cb ColumnBlockBuilder) Build() Block {
	return cb.block
}

type Column struct {
	Values []float64
}
