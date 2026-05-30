package middleware

import (
	"sync"
	"time"
)

type attemptRecord struct {
	count     int
	firstFail time.Time
	lockedUntil time.Time
}

type ShareBruteProtector struct {
	mu       sync.Mutex
	attempts map[string]*attemptRecord
	maxFails int
	lockDur  time.Duration
}

func NewShareBruteProtector(maxFails int, lockDuration time.Duration) *ShareBruteProtector {
	p := &ShareBruteProtector{
		attempts: make(map[string]*attemptRecord),
		maxFails: maxFails,
		lockDur:  lockDuration,
	}
	go p.cleanupLoop()
	return p
}

func (p *ShareBruteProtector) RecordFailed(token, ip string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := token + ":" + ip
	now := time.Now()
	rec, exists := p.attempts[key]
	if !exists {
		p.attempts[key] = &attemptRecord{count: 1, firstFail: now}
		return false
	}

	// Reset if lock duration passed
	if now.After(rec.lockedUntil) && rec.lockedUntil.After(time.Time{}) {
		rec.count = 1
		rec.firstFail = now
		rec.lockedUntil = time.Time{}
		return false
	}

	rec.count++
	if rec.count >= p.maxFails {
		rec.lockedUntil = now.Add(p.lockDur)
		return true // locked
	}
	return false
}

func (p *ShareBruteProtector) IsLocked(token, ip string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := token + ":" + ip
	rec, exists := p.attempts[key]
	if !exists {
		return false
	}
	if rec.lockedUntil.After(time.Time{}) && time.Now().Before(rec.lockedUntil) {
		return true
	}
	return false
}

func (p *ShareBruteProtector) Reset(token, ip string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.attempts, token+":"+ip)
}

func (p *ShareBruteProtector) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		p.mu.Lock()
		now := time.Now()
		for key, rec := range p.attempts {
			// Remove records older than 2x lock duration
			if now.After(rec.firstFail.Add(p.lockDur * 2)) {
				delete(p.attempts, key)
			}
		}
		p.mu.Unlock()
	}
}
