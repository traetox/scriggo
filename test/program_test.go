// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test

import (
	"io/fs"
	"reflect"
	"testing"

	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/internal/fstest"
	"github.com/open2b/scriggo/native"
)

type TypeStruct struct{}

func (t TypeStruct) Method() {}

// TestIssue403 executes a test the issue
// https://github.com/open2b/scriggo/issues/403. This issue cannot be tested
// with usual tests as it would require an execution environment that would be
// hard to reproduce without writing host code
func TestIssue403(t *testing.T) {
	t.Run("Method call on predefined variable", func(t *testing.T) {
		packages := native.CombinedLoader{
			native.Packages{
				"pkg": &native.DeclarationsPackage{
					PkgName: "pkg",
					Declarations: map[string]interface{}{
						"Value": &TypeStruct{},
					},
				},
			},
		}
		main := `
		package main

		import "pkg"

		func main() {
			pkg.Value.Method()
		}`
		fsys := fstest.Files{"main.go": main}
		program, err := scriggo.Build(fsys, &scriggo.BuildOptions{Packages: packages})
		if err != nil {
			t.Fatal(err)
		}
		err = program.Run(nil)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Method call on not-predefined variable", func(t *testing.T) {
		packages := native.CombinedLoader{
			native.Packages{
				"pkg": &native.DeclarationsPackage{
					PkgName: "pkg",
					Declarations: map[string]interface{}{
						"Type": reflect.TypeOf(new(TypeStruct)).Elem(),
					},
				},
			},
		}
		main := `
		package main

		import "pkg"
		
		func main() {
			t := pkg.Type{}
			t.Method()
		}`
		fsys := fstest.Files{"main.go": main}
		program, err := scriggo.Build(fsys, &scriggo.BuildOptions{Packages: packages})
		if err != nil {
			t.Fatal(err)
		}
		err = program.Run(nil)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Function that takes a struct as argument", func(t *testing.T) {
		packages := native.CombinedLoader{
			native.Packages{
				"pkg": &native.DeclarationsPackage{
					PkgName: "pkg",
					Declarations: map[string]interface{}{
						"F": func(s struct{}) {},
					},
				},
			},
		}
		main := `
	
		package main

		import "pkg"
		
		func main() {
			t := struct{}{}
			pkg.F(t)
		}
		`
		fsys := fstest.Files{"main.go": main}
		program, err := scriggo.Build(fsys, &scriggo.BuildOptions{Packages: packages})
		if err != nil {
			t.Fatal(err)
		}
		err = program.Run(nil)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Function taking an array", func(t *testing.T) {
		packages := native.CombinedLoader{
			native.Packages{
				"pkg": &native.DeclarationsPackage{
					PkgName: "pkg",
					Declarations: map[string]interface{}{
						"F": func(s [3]int) {},
					},
				},
			},
		}
		main := `
	
		package main

		import "pkg"
		
		func main() {
			a := [3]int{}
			pkg.F(a)
		}
		`
		fsys := fstest.Files{"main.go": main}
		program, err := scriggo.Build(fsys, &scriggo.BuildOptions{Packages: packages})
		if err != nil {
			t.Fatal(err)
		}
		err = program.Run(nil)
		if err != nil {
			t.Fatal(err)
		}
	})
}

// TestIssue309 executes a test the issue
// https://github.com/open2b/scriggo/issues/309.
func TestIssue309(t *testing.T) {
	t.Run("Add right position to 'imported and not used' errors", func(t *testing.T) {
		packages := native.CombinedLoader{
			native.Packages{
				"pkg": &native.DeclarationsPackage{
					PkgName: "pkg",
					Declarations: map[string]interface{}{
						"Value": &TypeStruct{},
					},
				},
			},
		}
		main := `
        package main

		import (
			"pkg"
		)

		func main() { }`
		fsys := fstest.Files{"main.go": main}
		_, err := scriggo.Build(fsys, &scriggo.BuildOptions{Packages: packages})
		if err == nil {
			t.Fatal("unexpected nil error")
		}
		err2, ok := err.(*scriggo.BuildError)
		if !ok {
			t.Fatalf("unexpected error %s, expecting a compiler error", err)
		}
		expectedPosition := scriggo.Position{Line: 5, Column: 4, Start: 37, End: 41}
		if err2.Position() != expectedPosition {
			t.Fatalf("unexpected position %#v, expecting %#v", err2.Position(), expectedPosition)
		}
	})
}

var compositeStructLiteralTests = []struct {
	fsys fs.FS
	pass bool
	err  string
}{{
	fsys: fstest.Files{
		"go.mod":  "module a.b\ngo 1.16",
		"main.go": `package main; import "a.b/p"; func main() { _ = p.S{5} }`,
		"p/p.go":  `package p; type S struct{ F int }`,
	},
	pass: true,
}, {
	fsys: fstest.Files{
		"go.mod":  "module a.b\ngo 1.16",
		"main.go": `package main; import "a.b/p"; func main() { _ = p.S{3, 5} }`,
		"p/p.go":  `package p; type S struct{ F, f int }`,
	},
	err: "main:1:56: implicit assignment of unexported field 'f' in p.S literal",
}, {
	fsys: fstest.Files{
		"go.mod":  "module a.b\ngo 1.16",
		"main.go": `package main; import "a.b/p"; func main() { _ = p.S{F: 3} }`,
		"p/p.go":  `package p; type T struct { F int }; type S struct{ T }`,
	},
	err: "main:1:53: cannot use promoted field T.F in struct literal of type p.S",
}}

// TestCompositeStructLiterals tests composite struct literals when the struct
// is defined in another package.
func TestCompositeStructLiterals(t *testing.T) {
	for _, cas := range compositeStructLiteralTests {
		_, err := scriggo.Build(cas.fsys, nil)
		if cas.pass {
			if err != nil {
				t.Fatalf("unexpected error: %q", err)
			}
		} else {
			if err == nil {
				t.Fatalf("expected error %q, got no error", cas.err)
			}
			if cas.err != err.Error() {
				t.Fatalf("expected error %q, got error %q", cas.err, err)
			}
		}
	}
}
