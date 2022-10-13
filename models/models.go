// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Models is a common place to store all models for use through flowgre.

package models

import (
	"context"
	"sync"
)

type Config struct {
	Server    string          `json:"server,omitempty"`
	DstPort   int             `json:"dst_port,omitempty"`
	Workers   int             `json:"workers,omitempty"`
	Delay     int             `json:"delay,omitempty"`
	WebIP     string          `json:"web_ip,omitempty"`
	WebPort   int             `json:"web_port,omitempty"`
	Web       bool            `json:"web,omitempty"`
	WaitGroup sync.WaitGroup  `json:"wait_group"`
	Context   context.Context `json:"context,omitempty"`
}

type WorkerStat struct {
	WorkerID  int    `json:"worker_id,omitempty"`
	SourceID  int    `json:"source_id,omitempty"`
	FlowsSent uint64 `json:"flows_sent,omitempty"`
	Cycles    uint64 `json:"cycles,omitempty"`
	BytesSent uint64 `json:"bytes_sent,omitempty"`
}

type StatTotals struct {
	FlowsSent uint64
	Cycles    uint64
	BytesSent uint64
}

type WorkerStats []WorkerStat

type Health struct {
	Status  string `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
}

type DashboardPage struct {
	Title       string             `json:"title,omitempty"`
	Comment     string             `json:"comment,omitempty"`
	HealthOut   Health             `json:"health_out"`
	ConfigOut   *Config            `json:"config_out"`
	StatsMapOut map[int]WorkerStat `json:"stats_map_out"`
	StatsTotal  StatTotals         `json:"stats_total"`
}
