// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package stats

import (
	"sync"
	"testing"
)

func TestRecordStatIncrValid(t *testing.T) {
	rs := RecordStat{}

	val := rs.IncrValid()
	if val != 1 {
		t.Errorf("expected 1 after first increment, got %d", val)
	}

	val = rs.IncrValid()
	if val != 2 {
		t.Errorf("expected 2 after second increment, got %d", val)
	}

	if rs.LoadValid() != 2 {
		t.Errorf("expected LoadValid() == 2, got %d", rs.LoadValid())
	}
}

func TestRecordStatIncrInvalid(t *testing.T) {
	rs := RecordStat{}

	val := rs.IncrInvalid()
	if val != 1 {
		t.Errorf("expected 1 after first increment, got %d", val)
	}

	if rs.LoadInvalid() != 1 {
		t.Errorf("expected LoadInvalid() == 1, got %d", rs.LoadInvalid())
	}
}

func TestRecordStatLoadZero(t *testing.T) {
	rs := RecordStat{}

	if rs.LoadValid() != 0 {
		t.Errorf("expected LoadValid() == 0, got %d", rs.LoadValid())
	}
	if rs.LoadInvalid() != 0 {
		t.Errorf("expected LoadInvalid() == 0, got %d", rs.LoadInvalid())
	}
}

func TestRecordStatConcurrentIncrements(t *testing.T) {
	rs := RecordStat{}

	const goroutines = 100
	const incrementsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				rs.IncrValid()
			}
		}()
	}

	wg.Wait()

	expected := uint64(goroutines * incrementsPerGoroutine)
	if rs.LoadValid() != expected {
		t.Errorf("expected %d valid counts, got %d", expected, rs.LoadValid())
	}
}

func TestRecordStatConcurrentMixedIncrements(t *testing.T) {
	rs := RecordStat{}

	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			if id%2 == 0 {
				rs.IncrValid()
			} else {
				rs.IncrInvalid()
			}
		}(i)
	}

	wg.Wait()

	valid := rs.LoadValid()
	invalid := rs.LoadInvalid()

	if valid+invalid != uint64(goroutines) {
		t.Errorf("expected total of %d, got %d (%d valid + %d invalid)",
			goroutines, valid+invalid, valid, invalid)
	}
}

func TestRecordStatDirectFieldAccess(t *testing.T) {
	rs := RecordStat{
		ValidCount:   10,
		InvalidCount: 5,
	}

	if rs.LoadValid() != 10 {
		t.Errorf("expected 10, got %d", rs.LoadValid())
	}
	if rs.LoadInvalid() != 5 {
		t.Errorf("expected 5, got %d", rs.LoadInvalid())
	}

	rs.IncrValid()
	rs.IncrInvalid()

	if rs.LoadValid() != 11 {
		t.Errorf("expected 11, got %d", rs.LoadValid())
	}
	if rs.LoadInvalid() != 6 {
		t.Errorf("expected 6, got %d", rs.LoadInvalid())
	}
}
