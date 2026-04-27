package counter

import (
	"context"
	"sync"

	"github.com/szymon/go-datastar-counter-demo/internal/validate"
)

type Store interface {
	Snapshot(ctx context.Context) (Snapshot, error)
	Apply(ctx context.Context, delta int) (Snapshot, error)
	Reset(ctx context.Context) (Snapshot, error)
}

type Hub struct {
	store Store
	mu    sync.RWMutex
	subs  map[chan Snapshot]struct{}
}

func NewHub(store Store) *Hub {
	return &Hub{
		store: store,
		subs:  make(map[chan Snapshot]struct{}),
	}
}

func (h *Hub) Snapshot(ctx context.Context) (Snapshot, error) {
	return h.store.Snapshot(ctx)
}

func (h *Hub) Subscribe(ctx context.Context) (<-chan Snapshot, func(), error) {
	snapshot, err := h.store.Snapshot(ctx)
	if err != nil {
		return nil, nil, err
	}

	ch := make(chan Snapshot, 8)
	ch <- snapshot

	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()

	unsubscribe := func() {
		h.mu.Lock()
		if _, ok := h.subs[ch]; ok {
			delete(h.subs, ch)
			close(ch)
		}
		h.mu.Unlock()
	}

	return ch, unsubscribe, nil
}

func (h *Hub) Change(ctx context.Context, delta int) (Snapshot, error) {
	if err := validate.Change(delta); err != nil {
		snapshot, snapErr := h.store.Snapshot(ctx)
		snapshot.Error = err.Error()
		return snapshot, snapErr
	}

	snapshot, err := h.store.Apply(ctx, delta)
	if err != nil {
		snapshot, snapErr := h.store.Snapshot(ctx)
		snapshot.Error = err.Error()
		return snapshot, snapErr
	}

	h.broadcast(snapshot)
	return snapshot, nil
}

func (h *Hub) Reset(ctx context.Context) (Snapshot, error) {
	snapshot, err := h.store.Reset(ctx)
	if err != nil {
		return snapshot, err
	}
	h.broadcast(snapshot)
	return snapshot, nil
}

func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		delete(h.subs, ch)
		close(ch)
	}
}

func (h *Hub) broadcast(snapshot Snapshot) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subs {
		select {
		case ch <- snapshot:
		default:
		}
	}
}
