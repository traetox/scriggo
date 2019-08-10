// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vm

import (
	"context"
	"sync"
)

type TraceFunc func(fn *Function, pc uint32, regs Registers)
type PrintFunc func(interface{})

// Env represents an execution environment.
type Env struct {

	// Only freeMemory, exited and exits fields can be changed after the vm
	// has been started and access to these three fields must be done with
	// this mutex.
	mu sync.Mutex

	ctx         context.Context // context.
	globals     []interface{}   // global variables.
	trace       TraceFunc       // trace function.
	print       PrintFunc       // custom print builtin.
	freeMemory  int             // free memory.
	limitMemory bool            // reports whether memory is limited.
	dontPanic   bool            // don't panic.
	exited      bool            // reports whether it is exited.
	exits       []func()        // exit functions.

}

// Alloc allocates, or if bytes is negative, deallocates memory. Alloc does
// nothing if there is no memory limit. If there is no free memory, Alloc
// panics with the OutOfMemory error.
func (env *Env) Alloc(bytes int) {
	if env.limitMemory {
		env.mu.Lock()
		free := env.freeMemory
		if free >= 0 {
			free -= int(bytes)
			env.freeMemory = free
		}
		env.mu.Unlock()
		if free < 0 {
			panic(ErrOutOfMemory)
		}
	}
}

// Context returns the context of the environment.
func (env *Env) Context() context.Context {
	return env.ctx
}

// ExitFunc calls f in its own goroutine after the execution of the
// environment is terminated.
func (env *Env) ExitFunc(f func()) {
	env.mu.Lock()
	if env.exited {
		go f()
	} else {
		env.exits = append(env.exits, f)
	}
	env.mu.Unlock()
	return
}

// FreeMemory returns the current free memory in bytes and true if the maximum
// memory has been limited. Otherwise returns zero and false.
//
// A negative value means that an out of memory error has been occurred and in
// this case bytes represents the number of bytes that were not available.
func (env *Env) FreeMemory() (bytes int, limitedMemory bool) {
	if env.limitMemory {
		env.mu.Lock()
		free := env.freeMemory
		env.mu.Unlock()
		return free, true
	}
	return 0, false
}
