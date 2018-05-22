package block

import (
	"time"
	"fmt"
)

type ColumnBlockBuilder struct {
	block *columnBlock
}

type columnBlock struct {
	columns [] column
	meta    BlockMetadata
}

func (c *columnBlock) Meta() BlockMetadata {
	return c.meta
}

func (c *columnBlock) StepIter() StepIter {
	return &colBlockIter{
		columns: c.columns,
		meta:    c.meta,
	}
}

// TODO: allow series iteration
func (c *columnBlock) SeriesIter() SeriesIter {
	return nil
}

// TODO: allow series iteration
func (c *columnBlock) SeriesMeta() []SeriesMeta {
	return nil
}

type colBlockIter struct {
	columns []column
	meta    BlockMetadata
	index   int
}

func (c *colBlockIter) Next() bool {
	return c.index < len(c.columns)
}

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

func (c colStep) Time() time.Time {
	return c.time
}

func (c colStep) Values() []float64 {
	return c.values
}

func NewColumnBlockBuilder(meta BlockMetadata) ColumnBlockBuilder {
	return ColumnBlockBuilder{
		block: &columnBlock{
			meta: meta,
		},
	}
}

func (cb ColumnBlockBuilder) AppendValue(index int, value float64) error {
	if len(cb.block.columns) <= index {
		return fmt.Errorf("index out of range for append: %d", index)
	}

	cb.block.columns[index].Values = append(cb.block.columns[index].Values, value)
	return nil
}

func (cb ColumnBlockBuilder) AddCols(num int) error {
	newCols := make([]column, num)
	cb.block.columns = append(cb.block.columns, newCols...)
	return nil
}

// TODO: Return an immutable copy
func (cb ColumnBlockBuilder) Build() Block {
	return cb.block
}

type column struct {
	Values []float64
}
