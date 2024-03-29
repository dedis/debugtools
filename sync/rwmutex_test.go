// This file is adapted from the GO sync package.
// It originally contains the following license:
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// GOMAXPROCS=10 go test

package sync

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"testing"
)

func parallelReader(m *RWMutex, clocked, cunlock, cdone chan bool) {
	m.RLock()
	clocked <- true
	<-cunlock
	m.RUnlock()
	cdone <- true
}

func doTestParallelReaders(numReaders, gomaxprocs int) {
	runtime.GOMAXPROCS(gomaxprocs)
	var m RWMutex
	clocked := make(chan bool)
	cunlock := make(chan bool)
	cdone := make(chan bool)
	for i := 0; i < numReaders; i++ {
		go parallelReader(&m, clocked, cunlock, cdone)
	}
	// Wait for all parallel RLock()s to succeed.
	for i := 0; i < numReaders; i++ {
		<-clocked
	}
	for i := 0; i < numReaders; i++ {
		cunlock <- true
	}
	// Wait for the goroutines to finish.
	for i := 0; i < numReaders; i++ {
		<-cdone
	}
}

func parallelReaders() {
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(-1))
	doTestParallelReaders(1, 4)
	doTestParallelReaders(3, 4)
	doTestParallelReaders(4, 2)
}

func TestParallelReadersDebugOff(_ *testing.T) {
	DebugIsOn = false
	parallelReaders()
}

func TestParallelReadersDebugOn(_ *testing.T) {
	DebugIsOn = true
	parallelReaders()
}

func reader(rwm *RWMutex, numIterations int, activity *int32, cdone chan bool) {
	for i := 0; i < numIterations; i++ {
		rwm.RLock()
		n := atomic.AddInt32(activity, 1)
		if n < 1 || n >= 10000 {
			rwm.RUnlock()
			panic(fmt.Sprintf("wlock(%d)\n", n))
		}
		atomic.AddInt32(activity, -1)
		rwm.RUnlock()
	}
	cdone <- true
}

func writer(rwm *RWMutex, numIterations int, activity *int32, cdone chan bool) {
	for i := 0; i < numIterations; i++ {
		rwm.Lock()
		n := atomic.AddInt32(activity, 10000)
		if n != 10000 {
			rwm.Unlock()
			panic(fmt.Sprintf("wlock(%d)\n", n))
		}
		atomic.AddInt32(activity, -10000)
		rwm.Unlock()
	}
	cdone <- true
}

func HammerRWMutex(gomaxprocs, numReaders, numIterations int) {
	runtime.GOMAXPROCS(gomaxprocs)
	// Number of active readers + 10000 * number of active writers.
	var activity int32
	var rwm RWMutex
	cdone := make(chan bool)
	go writer(&rwm, numIterations, &activity, cdone)
	var i int
	for i = 0; i < numReaders/2; i++ {
		go reader(&rwm, numIterations, &activity, cdone)
	}
	go writer(&rwm, numIterations, &activity, cdone)
	for ; i < numReaders; i++ {
		go reader(&rwm, numIterations, &activity, cdone)
	}
	// Wait for the 2 writers and all readers to finish.
	for i := 0; i < 2+numReaders; i++ {
		<-cdone
	}
}

func rwMutex(t *testing.T) {
	var m RWMutex

	m.Lock()
	if m.TryLock() {
		t.Fatalf("TryLock succeeded with mutex locked")
	}
	if m.TryRLock() {
		t.Fatalf("TryRLock succeeded with mutex locked")
	}
	m.Unlock()

	if !m.TryLock() {
		t.Fatalf("TryLock failed with mutex unlocked")
	}
	m.Unlock()

	if !m.TryRLock() {
		t.Fatalf("TryRLock failed with mutex unlocked")
	}
	if !m.TryRLock() {
		t.Fatalf("TryRLock failed with mutex rlocked")
	}
	if m.TryLock() {
		t.Fatalf("TryLock succeeded with mutex rlocked")
	}
	m.RUnlock()
	m.RUnlock()

	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(-1))
	n := 1000
	if testing.Short() {
		n = 5
	}
	HammerRWMutex(1, 1, n)
	HammerRWMutex(1, 3, n)
	HammerRWMutex(1, 10, n)
	HammerRWMutex(4, 1, n)
	HammerRWMutex(4, 3, n)
	HammerRWMutex(4, 10, n)
	HammerRWMutex(10, 1, n)
	HammerRWMutex(10, 3, n)
	HammerRWMutex(10, 10, n)
	HammerRWMutex(10, 5, n)
}

func TestRWMutexDebugOff(t *testing.T) {
	DebugIsOn = false
	rwMutex(t)
}

func TestRWMutexDebugOn(t *testing.T) {
	DebugIsOn = true
	rwMutex(t)
}
