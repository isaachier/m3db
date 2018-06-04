package transform

import (
	"time"

	"github.com/m3db/m3db/src/coordinator/block"
	"github.com/m3db/m3db/src/coordinator/parser"
)

// Options to create transform nodes
type Options struct {
	Now time.Time
}

// OpNode represents the execution node
type OpNode interface {
	Process(ID parser.NodeID, block block.Block) error
}
