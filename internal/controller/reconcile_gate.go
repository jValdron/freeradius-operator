package controller

import (
	"sync"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
)

type reconcileGate struct {
	mu   sync.Mutex
	keys map[string]struct{}
}

func (g *reconcileGate) run(key string, fn func() (ctrl.Result, error)) (ctrl.Result, error) {
	g.mu.Lock()
	if g.keys == nil {
		g.keys = map[string]struct{}{}
	}
	if _, busy := g.keys[key]; busy {
		g.mu.Unlock()
		return ctrl.Result{RequeueAfter: 500 * time.Millisecond}, nil
	}
	g.keys[key] = struct{}{}
	g.mu.Unlock()

	defer func() {
		g.mu.Lock()
		delete(g.keys, key)
		g.mu.Unlock()
	}()

	return fn()
}
