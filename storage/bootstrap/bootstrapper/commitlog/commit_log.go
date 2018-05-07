// Copyright (c) 2016 Uber Technologies, Inc.
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

package commitlog

import (
	"github.com/m3db/m3db/persist/fs"
	"github.com/m3db/m3db/storage/bootstrap"
	"github.com/m3db/m3db/storage/bootstrap/bootstrapper"
)

const (
	// CommitLogBootstrapperName is the name of the commit log bootstrapper.
	CommitLogBootstrapperName = "commitlog"
)

type commitLogBootstrapperProvider struct {
	opts       Options
	inspection fs.Inspection
	next       bootstrap.BootstrapperProvider
}

// NewCommitLogBootstrapperProvider creates a new bootstrapper provider
// to bootstrap from commit log files.
func NewCommitLogBootstrapperProvider(
	opts Options,
	inspection fs.Inspection,
	next bootstrap.BootstrapperProvider,
) (bootstrap.BootstrapperProvider, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}
	return commitLogBootstrapperProvider{
		opts: opts,
		next: next,
	}, nil
}

func (p commitLogBootstrapperProvider) Provide() bootstrap.Bootstrapper {
	var (
		src  = newCommitLogSource(p.opts, p.inspection)
		b    = &commitLogBootstrapper{}
		next bootstrap.Bootstrapper
	)
	if p.next != nil {
		next = p.next.Provide()
	}
	b.Bootstrapper = bootstrapper.NewBaseBootstrapper(b.String(),
		src, p.opts.ResultOptions(), next)
	return b
}

func (p commitLogBootstrapperProvider) String() string {
	return CommitLogBootstrapperName
}

type commitLogBootstrapper struct {
	bootstrap.Bootstrapper
}

func (*commitLogBootstrapper) String() string {
	return CommitLogBootstrapperName
}
