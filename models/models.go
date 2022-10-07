// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Models is a common place to store all models for use through flowgre.

package models

import (
	"context"
	"sync"
)

type Config struct {
	Server    string
	DstPort   int
	Workers   int
	Delay     int
	WebIP     string
	WebPort   int
	Web       bool
	WaitGroup sync.WaitGroup
	Context   context.Context
}

type WorkerStat struct {
	WorkerID  int
	SourceID  int
	FlowsSent uint64
	Cycles    uint64
	BytesSent uint64
}

type WorkerStats []WorkerStat

type Health struct {
	Status  string
	Message string
}
