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
)

// ColumnBlockBuilder builds a block optimized for column iteration
type ColumnBlockBuilder struct {
	block *columnBlock
}

type columnBlock struct {
	columns [] column
	meta    Metadata
}

// Meta returns the metadata for the block
func (c *columnBlock) Meta() Metadata {
	return c.meta
}

// StepIter returns a StepIterator
func (c *columnBlock) StepIter() StepIter {
	return &colBlockIter{
		columns: c.columns,
		meta:    c.meta,
	}
}

// TODO: allow series iteration
// SeriesIter returns a SeriesIterator
func (c *columnBlock) SeriesIter() SeriesIter {
	return nil
}

// TODO: allow series iteration
// SeriesMeta returns the metadata for each series in the block
func (c *columnBlock) SeriesMeta() []SeriesMeta {
	return nil
}

// Close frees up any resources
func (c *columnBlock) Close() error {
	return nil
}


type colBlockIter struct {
	columns []column
	meta    Metadata
	index   int
}

// Next returns true if iterator has more values remaining
func (c *colBlockIter) Next() bool {
	return c.index < len(c.columns)
}

// Current returns the current step
func (c *colBlockIter) Current() Step {
	if !c.Next() {
		panic("current called without next")
	}

	col := c.columns[c.index]
	t, err := calcTime(c.meta.Bounds, c.index)
	if err != nil {
		panic(err)
	}

	c.index++
	return colStep{
		time:   t,
		values: col.Values,
	}
}

// Len returns the total length ignoring current iterator position
func (c *colBlockIter) Len() int {
	return len(c.columns)
}

func calcTime(bounds Bounds, index int) (time.Time, error) {
	step := bounds.StepSize
	t := bounds.Start.Add(time.Duration(index) * step)
	if t.After(bounds.End) {
		return time.Time{}, fmt.Errorf("out of bounds, %d", index)
	}

	return t, nil
}

type colStep struct {
	time   time.Time
	values []float64
}

// Time for the step
func (c colStep) Time() time.Time {
	return c.time
}

// Values for the column
func (c colStep) Values() []float64 {
	return c.values
}

// NewColumnBlockBuilder creates a new column block builder
func NewColumnBlockBuilder(meta Metadata) ColumnBlockBuilder {
	return ColumnBlockBuilder{
		block: &columnBlock{
			meta: meta,
		},
	}
}

// AppendValue adds a value to a column at index
func (cb ColumnBlockBuilder) AppendValue(index int, value float64) error {
	if len(cb.block.columns) <= index {
		return fmt.Errorf("index out of range for append: %d", index)
	}

	cb.block.columns[index].Values = append(cb.block.columns[index].Values, value)
	return nil
}

// AddCols adds new columns
func (cb ColumnBlockBuilder) AddCols(num int) error {
	newCols := make([]column, num)
	cb.block.columns = append(cb.block.columns, newCols...)
	return nil
}

// Build extracts the block
// TODO: Return an immutable copy
func (cb ColumnBlockBuilder) Build() Block {
	return cb.block
}

type column struct {
	Values []float64
}
