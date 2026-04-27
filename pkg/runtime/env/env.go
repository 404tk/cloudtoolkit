// Package env carries the per-run configuration envelope that used to live as
// loose package-level globals in utils/const.go.
//
// Three access patterns are supported:
//
//  1. Context-attached (preferred) — call sites that already have a
//     context.Context use env.From(ctx) to retrieve the run's config. The
//     headless layer can attach a per-run *Env via env.With(ctx, env), so
//     `ctk run --timeout 5m` does not pollute REPL state or other concurrent
//     runs.
//
//  2. Process-active singleton (fallback) — capability methods that lack
//     ctx (EventDump, parseRDSAccount) read env.Active(). Replaced atomically
//     so concurrent reads see a consistent snapshot, unlike the previous
//     unsynchronised globals.
//
//  3. Tests — env.SetActiveForTest pins a value and registers a cleanup that
//     restores the previous active env, so 94 unit tests can keep using
//     ListPolicies / Cloudlist / RDSAccount overrides without leaking state
//     across t.Parallel boundaries.
//
// From(ctx) prefers an attached env when present, falling back to Active(),
// finally to a zero-valued default. Callers therefore never have to nil-check.
package env

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// Env is the per-run configuration envelope. Built once by the REPL or
// headless layer at startup; flows through context to providers and payloads.
//
// All fields are read-only after construction. Callers must clone before
// mutating.
type Env struct {
	LogEnable    bool
	ListPolicies bool
	LogDir       string
	Cloudlist    []string
	IAMUserCheck string
	RDSAccount   string
	RunTimeout   time.Duration
}

// Clone returns a deep copy. Use when constructing a per-run override so the
// active env is not aliased.
func (e *Env) Clone() *Env {
	if e == nil {
		return &Env{}
	}
	cp := *e
	if e.Cloudlist != nil {
		cp.Cloudlist = append([]string(nil), e.Cloudlist...)
	}
	return &cp
}

// Default returns a zero-valued Env with sensible fallbacks. Used when neither
// ctx nor active singleton has anything attached (test environments, library
// embedding without InitConfig).
func Default() *Env {
	return &Env{
		RunTimeout: 10 * time.Minute,
	}
}

type ctxKey struct{}

// With attaches env to ctx. Subsequent From(ctx) calls in the resulting tree
// see the supplied value. Passing nil is a no-op so callers can safely chain.
func With(ctx context.Context, env *Env) context.Context {
	if env == nil {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, env)
}

// From retrieves the env attached to ctx. Falls back to Active(), then to
// Default() — never returns nil.
func From(ctx context.Context) *Env {
	if ctx != nil {
		if v, ok := ctx.Value(ctxKey{}).(*Env); ok && v != nil {
			return v
		}
	}
	if v := active.Load(); v != nil {
		return v
	}
	return Default()
}

// active holds the process-active env. atomic.Pointer guarantees torn reads
// are impossible even without an explicit mutex; struct fields are immutable
// so swapping the whole pointer is enough.
var active atomic.Pointer[Env]

// SetActive replaces the process-active env. cmd/main.go calls this once
// after runner.InitConfig; nothing else should call it at runtime. Pass nil
// to clear (Active reverts to Default).
func SetActive(e *Env) {
	if e == nil {
		active.Store(nil)
		return
	}
	active.Store(e)
}

// Active returns the process-active env. Falls back to Default if nothing
// was set. Never returns nil.
func Active() *Env {
	if v := active.Load(); v != nil {
		return v
	}
	return Default()
}

// SetActiveForTest pins env as the active singleton and registers a cleanup
// that restores the prior value when the test ends. Use when a test needs
// to flip a single flag (ListPolicies, Cloudlist subset, ...) without
// fighting other tests.
func SetActiveForTest(t testing.TB, e *Env) {
	t.Helper()
	prev := active.Load()
	active.Store(e)
	t.Cleanup(func() { active.Store(prev) })
}
