// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package stats

import "sync/atomic"

// RecordStat tracks valid and invalid flow records atomically.
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
