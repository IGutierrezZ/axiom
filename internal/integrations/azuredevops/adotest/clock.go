package adotest

import (
	"context"
	"sync"
	"time"
)

// epoch es el instante inicial del reloj manual: redondo al segundo, para que las fechas HTTP de
// Retry-After (resolución de un segundo) coincidan con los segundos del reloj.
var epoch = time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)

// Clock es un reloj manual que implementa azuredevops.Clock. Sleep no duerme: registra la duración,
// ejecuta el gancho opcional, devuelve ctx.Err() si el contexto venció y avanza el tiempo virtual.
type Clock struct {
	mu     sync.Mutex
	now    time.Time
	sleeps []time.Duration
	hook   func(d time.Duration)
}

// NewClock devuelve un reloj manual en su instante inicial.
func NewClock() *Clock { return &Clock{now: epoch} }

// Now devuelve el tiempo virtual.
func (c *Clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Advance adelanta el tiempo virtual.
func (c *Clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// OnSleep instala un gancho que se ejecuta en cada Sleep antes de comprobar el contexto. Sirve para
// cancelar el contexto «durante» una espera.
func (c *Clock) OnSleep(hook func(d time.Duration)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hook = hook
}

// Sleep registra d, ejecuta el gancho, devuelve ctx.Err() si el contexto venció (sin avanzar el
// tiempo) y, si no, avanza el tiempo virtual d sin dormir.
func (c *Clock) Sleep(ctx context.Context, d time.Duration) error {
	c.mu.Lock()
	c.sleeps = append(c.sleeps, d)
	hook := c.hook
	c.mu.Unlock()

	if hook != nil {
		hook(d)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	c.Advance(d)
	return nil
}

// Sleeps devuelve las esperas pedidas, en orden.
func (c *Clock) Sleeps() []time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]time.Duration(nil), c.sleeps...)
}

// Slept devuelve la suma de las esperas pedidas.
func (c *Clock) Slept() time.Duration {
	var total time.Duration
	for _, d := range c.Sleeps() {
		total += d
	}
	return total
}
