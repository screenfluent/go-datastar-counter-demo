package store

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/szymon/go-datastar-counter-demo/internal/counter"
)

type Memory struct {
	mu    sync.Mutex
	value int32
}

func NewMemory() *Memory {
	return &Memory{}
}

func (m *Memory) Snapshot(context.Context) (counter.Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.snapshotLocked("memory"), nil
}

func (m *Memory) Apply(_ context.Context, delta int) (counter.Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	next := m.value + int32(delta)
	if next < 0 {
		snapshot := m.snapshotLocked("memory")
		snapshot.Error = "licznik nie moze spasc ponizej zera"
		return snapshot, errors.New(snapshot.Error)
	}

	m.value = next
	return m.snapshotLocked("memory"), nil
}

func (m *Memory) Reset(context.Context) (counter.Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.value = 0
	return m.snapshotLocked("memory"), nil
}

func (m *Memory) snapshotLocked(source string) counter.Snapshot {
	return counter.Snapshot{
		Value:     m.value,
		UpdatedAt: time.Now().UTC(),
		Source:    source,
	}
}
