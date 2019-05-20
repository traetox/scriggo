// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/tools/go/loader"
)

var goKeywords = []string{
	"break", "default", "func", "interface", "select", "case", "defer",
	"go", "map", "struct", "chan", "else", "goto", "package",
	"switch", "const", "fallthrough", "if", "range",
	"type", "continue", "for", "import", "return", "var",
}

var pkgNamesToPkgPaths = map[string]string{}

// uniquePackageName generates an unique package name for every package path.
func uniquePackageName(pkgPath string) string {
	pkgName := filepath.Base(pkgPath)
	done := false
	for !done {
		done = true
		cachePath, ok := pkgNamesToPkgPaths[pkgName]
		if ok && cachePath != pkgPath {
			done = false
			pkgName += "_"
		}
	}
	for _, goKwd := range goKeywords {
		if goKwd == pkgName {
			pkgName = "_" + pkgName + "_"
		}
	}
	pkgNamesToPkgPaths[pkgName] = pkgPath
	return pkgName
}

// goPackageToDeclarations navigates the package pkgPath and returns a map
// containing the exported declarations.
func goPackageToDeclarations(pkgPath string) (map[string]string, error) {

	out := make(map[string]string)

	pkgBase := uniquePackageName(pkgPath)
	config := loader.Config{}
	config.Import(pkgPath)
	program, err := config.Load()
	if err != nil {
		return nil, err
	}
	pkgInfo := program.Package(pkgPath)
	for _, file := range pkgInfo.Files {
		for _, decl := range file.Decls {
			if funcDecl, ok := decl.(*ast.FuncDecl); ok {
				if !isExported(funcDecl.Name.Name) {
					continue
				}
				if funcDecl.Recv != nil { // is a method.
					continue
				}
				out[funcDecl.Name.Name] = pkgBase + "." + funcDecl.Name.Name
			} else if genDecl, ok := decl.(*ast.GenDecl); ok {
				switch genDecl.Tok {
				case token.CONST, token.VAR:
					for _, spec := range genDecl.Specs {
						if valueSpec, ok := spec.(*ast.ValueSpec); ok {
							for i, name := range valueSpec.Names {
								if !isExported(name.Name) {
									continue
								}
								if i > len(valueSpec.Values)-1 {
									continue
								}
								if genDecl.Tok == token.CONST {
									expr := valueSpec.Values[i]
									typ := "nil"
									isTyped := valueSpec.Type != nil
									if isTyped {
										typ = "reflect.TypeOf(" + pkgBase + "." + name.Name + ")"
									}
									expression := strconv.Quote(pkgInfo.Types[expr].Value.ExactString())
									out[name.Name] = fmt.Sprintf("native.Constant(%s, %s)", expression, typ)
								} else {
									out[name.Name] = "&" + pkgBase + "." + name.Name
								}
							}
						}
					}
				case token.TYPE:
					for _, spec := range genDecl.Specs {
						spec := spec.(*ast.TypeSpec)
						if !isExported(spec.Name.Name) {
							continue
						}
						out[spec.Name.Name] = "reflect.TypeOf(new(" + pkgBase + "." + spec.Name.Name + ")).Elem()"
					}
				}
			}
		}
	}

	return out, nil
}

func isExported(name string) bool {
	return unicode.Is(unicode.Lu, []rune(name)[0])
}

var outputSkeleton = `[generatedWarning]

package [pkgName]

import (
	[explicitImports]
)

import "scrigo"

func init() {
	[customVariableName] = map[string]*native.GoPackage{
		[pkgContent]
	}
}
`

// generatePackages generate all packages imported in sourceFile, creating the
// package pkgName and storing them in customVariableName.
func generatePackages(pkgs []string, sourceFile, customVariableName, pkgName string) string {
	explicitImports := ""
	for _, p := range pkgs {
		explicitImports += uniquePackageName(p) + `"` + p + `"` + "\n"
	}

	pkgContent := strings.Builder{}
	for _, p := range pkgs {
		out := generatePackage(p)
		pkgContent.WriteString(out)
	}

	r := strings.NewReplacer(
		"[generatedWarning]", "// Code generated by scrigo-generate, based on file \""+sourceFile+"\". DO NOT EDIT.",
		"[pkgName]", pkgName,
		"[explicitImports]", explicitImports,
		"[customVariableName]", customVariableName,
		"[pkgContent]", pkgContent.String(),
	)
	return r.Replace(outputSkeleton)
}

// generatePackage generates package pkgPath.
func generatePackage(pkgPath string) string {
	pkg, err := importer.Default().Import(pkgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "importer error: %s\n", err)
		return ""
	}

	decls, err := goPackageToDeclarations(pkgPath)
	if err != nil {
		panic(err)
	}

	// Sorts declarations.
	names := make([]string, 0, len(decls))
	for name := range decls {
		names = append(names, name)
	}
	sort.Strings(names)

	pkgContent := strings.Builder{}
	for _, name := range names {
		pkgContent.WriteString(`"` + name + `": ` + decls[name] + ",\n")
	}

	skel := `
		"[pkgPath]": &native.GoPackage{
			Name: "[pkg.Name()]",
			Declarations: map[string]interface{}{
				[pkgContent]
			},
		},`

	repl := strings.NewReplacer(
		"[pkgPath]", pkgPath,
		"[pkgContent]", pkgContent.String(),
		"[pkg.Name()]", pkg.Name(),
	)

	return repl.Replace(skel)
}
