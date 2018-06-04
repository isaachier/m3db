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

package seriesiter

import (
	"time"

	"github.com/m3db/m3db/src/dbnode/encoding"
	m3ts "github.com/m3db/m3db/src/dbnode/ts"
	"github.com/m3db/m3x/ident"
	xtime "github.com/m3db/m3x/time"

	"github.com/golang/mock/gomock"
)

// GenerateSingleSampleTagIterator generates a new tag iterator
func GenerateSingleSampleTagIterator(ctrl *gomock.Controller, tag ident.Tag) ident.TagIterator {
	mockTagIterator := ident.NewMockTagIterator(ctrl)
	mockTagIterator.EXPECT().Remaining().Return(1)
	mockTagIterator.EXPECT().Next().Return(true).MaxTimes(1)
	mockTagIterator.EXPECT().Current().Return(tag)
	mockTagIterator.EXPECT().Next().Return(false)
	mockTagIterator.EXPECT().Err().Return(nil)
	mockTagIterator.EXPECT().Close()

	return mockTagIterator
}

// GenerateTag generates a new tag
func GenerateTag() ident.Tag {
	return ident.Tag{
		Name:  ident.StringID("foo"),
		Value: ident.StringID("bar"),
	}
}

// NewMockSeriesIters generates a new mock series iters
func NewMockSeriesIters(ctrl *gomock.Controller, tags ident.Tag, len int) encoding.SeriesIterators {
	iteratorList := make([]encoding.SeriesIterator, 0, len)
	for i := 0; i < len; i++ {
		mockIter := encoding.NewMockSeriesIterator(ctrl)
		mockIter.EXPECT().Next().Return(true).MaxTimes(2)
		mockIter.EXPECT().Next().Return(false)
		mockIter.EXPECT().Current().Return(m3ts.Datapoint{Timestamp: time.Now(), Value: 10}, xtime.Millisecond, nil)
		mockIter.EXPECT().Current().Return(m3ts.Datapoint{Timestamp: time.Now(), Value: 10}, xtime.Millisecond, nil)
		mockIter.EXPECT().ID().Return(ident.StringID("foo"))
		mockIter.EXPECT().Tags().Return(GenerateSingleSampleTagIterator(ctrl, tags))

		iteratorList = append(iteratorList, mockIter)
	}

	mockIters := encoding.NewMockSeriesIterators(ctrl)
	mockIters.EXPECT().Iters().Return(iteratorList)
	mockIters.EXPECT().Len().Return(len)
	mockIters.EXPECT().Close()

	return mockIters
}
