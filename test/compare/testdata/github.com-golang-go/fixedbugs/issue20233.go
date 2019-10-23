// errorcheck

// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Issue 20233: panic while formatting an error message

package main

var f = func(...A) // ERROR `syntax error: missing function body`

var f = func(...A) { } // ERROR `undefined: A`

func main() { }
