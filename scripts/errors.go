// Copyright 2019 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scripts

import (
	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/internal/compiler"
	"github.com/open2b/scriggo/internal/runtime"
)

// BuildError represents an error occurred building a script.
type BuildError struct {
	err compiler.Error
}

// Error returns a string representation of the error.
func (err *BuildError) Error() string {
	return err.err.Error()
}

// Path returns the path of the file where the error occurred.
func (err *BuildError) Path() string {
	return err.err.Path()
}

// Position returns the position in the file where the error occurred.
func (err *BuildError) Position() scriggo.Position {
	pos := err.err.Position()
	return scriggo.Position{Line: pos.Line, Column: pos.Column, Start: pos.Start, End: pos.End}
}

// Message returns the error message.
func (err *BuildError) Message() string {
	return err.err.Message()
}

// PanicError represents the error that occurs when an executed script calls
// the panic built-in and the panic is not recovered.
type PanicError struct {
	p *runtime.PanicError
}

// Error returns all currently active panics as a string.
//
// To print only the message, use the String method instead.
func (p *PanicError) Error() string {
	return p.p.Error()
}

// Message returns the panic message.
func (p *PanicError) Message() interface{} {
	return p.p.Message()
}

// Next returns the next panic in the chain.
func (p *PanicError) Next() *PanicError {
	return &PanicError{p.p.Next()}
}

// Recovered reports whether it has been recovered.
func (p *PanicError) Recovered() bool {
	return p.p.Recovered()
}

// String returns the panic message as a string.
func (p *PanicError) String() string {
	return p.p.String()
}

// Path returns the path of the file that panicked.
func (p *PanicError) Path() string {
	return p.p.Path()
}

// Position returns the position in file where the panic occurred.
func (p *PanicError) Position() scriggo.Position {
	pos := p.p.Position()
	return scriggo.Position{Line: pos.Line, Column: pos.Column, Start: pos.Start, End: pos.End}
}
