package storage

import (
	"github.com/m3db/m3db/src/coordinator/ts"
	"github.com/m3db/m3db/src/coordinator/block"
	"time"
)

func FetchResultToBlockResult(result *FetchResult, query *FetchQuery) (block.Result, error) {
	multiBlock, err := newMultiSeriesBlock(result.SeriesList, query)
	if err != nil {
		return block.Result{}, err
	}

	return block.Result{
		Blocks: []block.Block{multiBlock},
	}, nil
}

type multiSeriesBlock struct {
	seriesList ts.SeriesList
	meta       block.Metadata
}

func newMultiSeriesBlock(seriesList ts.SeriesList, query *FetchQuery) (multiSeriesBlock, error) {
	resolution, err := seriesList.Resolution()
	if err != nil {
		return multiSeriesBlock{}, err
	}

	meta := block.Metadata{
		Bounds: block.Bounds{
			Start:    query.Start,
			End:      query.End,
			StepSize: resolution,
		},
	}
	return multiSeriesBlock{seriesList: seriesList, meta: meta}, nil
}

func (m multiSeriesBlock) Meta() block.Metadata {
	return m.meta
}

func (m multiSeriesBlock) StepIter() block.StepIter {
	return &multiSeriesBlockStepIter{block: m}
}

func (m multiSeriesBlock) SeriesIter() block.SeriesIter {
	return &multiSeriesBlockSeriesIter{block: m}
}

func (m multiSeriesBlock) SeriesMeta() []block.SeriesMeta {
	metas := make([]block.SeriesMeta, len(m.seriesList))
	for i, s := range m.seriesList {
		metas[i].Tags = s.Tags
	}

	return metas
}

type multiSeriesBlockStepIter struct {
	block multiSeriesBlock
	index int
}

func (m *multiSeriesBlockStepIter) Next() bool {
	if len(m.block.seriesList) == 0 {
		return false
	}

	return m.index < m.block.seriesList[0].Values().Len()
}

func (m *multiSeriesBlockStepIter) Current() block.Step {
	t := m.block.meta.Bounds.Start.Add(time.Duration(m.index) * m.block.meta.Bounds.StepSize)
	values := make([]float64, len(m.block.seriesList))
	for i, s := range m.block.seriesList {
		values[i] = s.Values().ValueAt(i)
	}

	m.index++
	return colStep{
		t:      t,
		values: values,
	}
}

func (m *multiSeriesBlockStepIter) Len() int {
	if len(m.block.seriesList) == 0 {
		return 0
	}

	return m.block.seriesList[0].Values().Len()
}

type colStep struct {
	t      time.Time
	values []float64
}

func (c colStep) Time() time.Time {
	return c.t
}

func (c colStep) Values() []float64 {
	return c.values
}

type multiSeriesBlockSeriesIter struct {
	block multiSeriesBlock
	index int
}

func (m *multiSeriesBlockSeriesIter) Next() bool {
	return m.index < len(m.block.seriesList)
}

func (m *multiSeriesBlockSeriesIter) Current() block.Series {
	s := m.block.seriesList[m.index]
	values := make([]float64, s.Len())
	for i := 0; i < s.Len(); i++ {
		values[i] = s.Values().ValueAt(i)
	}
	return block.NewSeries(values, m.block.meta.Bounds)
}
