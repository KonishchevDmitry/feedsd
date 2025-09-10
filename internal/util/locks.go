package util

import (
	"sync"
)

type GuardedLock struct {
	lock sync.Mutex
}

func (l *GuardedLock) Guard() LockGuard {
	return MakeLockGuard(&l.lock)
}

func (l *GuardedLock) Lock() LockGuard {
	lock := l.Guard()
	lock.Lock()
	return lock //nolint:govet
}

type LockGuard struct {
	lock   sync.Locker
	locked bool
}

func MakeLockGuard(lock sync.Locker) LockGuard {
	return LockGuard{lock: lock}
}

func (l *LockGuard) Lock() {
	l.lock.Lock()
	l.locked = true
}

func (l *LockGuard) Unlock() {
	l.lock.Unlock()
	l.locked = false
}

func (l *LockGuard) UnlockIfLocked() {
	if l.locked {
		l.Unlock()
	}
}
