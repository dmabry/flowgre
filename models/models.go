// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Models is a common place to store all models for use through flowgre.

package models

import "sync/atomic"

type Config struct {
	Server   string `json:"server,omitempty"`
	DstPort  int    `json:"dst_port,omitempty"`
	SrcRange string `json:"src_range,omitempty"`
	DstRange string `json:"dst_range,omitempty"`
	Workers  int    `json:"workers,omitempty"`
	Delay    int    `json:"delay,omitempty"`
	WebIP    string `json:"web_ip,omitempty"`
	WebPort  int    `json:"web_port,omitempty"`
	Web      bool   `json:"web,omitempty"`
}

type WorkerStat struct {
	WorkerID  int    `json:"worker_id,omitempty"`
	SourceID  int    `json:"source_id,omitempty"`
	FlowsSent uint64 `json:"flows_sent,omitempty"`
	Cycles    uint64 `json:"cycles,omitempty"`
	BytesSent uint64 `json:"bytes_sent,omitempty"`
}

type RecordStat struct {
	ValidCount   uint64
	InvalidCount uint64
}

// IncrValid atomically increments ValidCount and returns the new value.
func (rs *RecordStat) IncrValid() uint64 {
	return atomic.AddUint64(&rs.ValidCount, 1)
}

// IncrInvalid atomically increments InvalidCount and returns the new value.
func (rs *RecordStat) IncrInvalid() uint64 {
	return atomic.AddUint64(&rs.InvalidCount, 1)
}

// LoadValid atomically loads ValidCount.
func (rs *RecordStat) LoadValid() uint64 {
	return atomic.LoadUint64(&rs.ValidCount)
}

// LoadInvalid atomically loads InvalidCount.
func (rs *RecordStat) LoadInvalid() uint64 {
	return atomic.LoadUint64(&rs.InvalidCount)
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
