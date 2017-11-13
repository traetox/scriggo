//
// Copyright (c) 2016-2017 Open2b Software Snc. All Rights Reserved.
//

// Package exec fornisce i metodi per eseguire gli alberi dei template.
package exec

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"unicode"

	"open2b/template/ast"
	"open2b/template/types"
)

type Error struct {
	Path string
	Pos  ast.Position
	Err  error
}

func (e *Error) Error() string {
	return fmt.Sprintf("template: %s at %q %s", e.Err, e.Path, e.Pos)
}

// errBreak è ritornato dall'esecuzione dello statement "break".
// Viene gestito dallo statement "for" più interno.
var errBreak = errors.New("break is not in a loop")

// errContinue è ritornato dall'esecuzione dello statement "break".
// Viene gestito dallo statement "for" più interno.
var errContinue = errors.New("continue is not in a loop")

// scope di variabili
type scope map[string]interface{}

var scopeType = reflect.TypeOf(scope{})

type Env struct {
	tree    *ast.Tree
	version string
}

// NewEnv ritorna un ambiente di esecuzione per l'albero tree.
func NewEnv(tree *ast.Tree, version string) *Env {
	if tree == nil {
		panic("template: tree is nil")
	}
	return &Env{tree, version}
}

// Execute esegue l'albero tree e scrive il risultato su wr.
// Le variabili in vars sono definite nell'ambiente durante l'esecuzione.
//
// vars può:
//
//   - avere map[string]interface{} come underlying type
//   - essere una struct
//   - essere un reflect.Value il cui valore concreto soddisfa uno dei precedenti
//   - essere nil
//
func (env *Env) Execute(wr io.Writer, vars interface{}) error {

	if wr == nil {
		return errors.New("template/exec: wr is nil")
	}

	globals, err := varsToScope(vars, env.version)
	if err != nil {
		return err
	}

	s := state{
		path:     env.tree.Path,
		vars:     []scope{builtins, globals, {}},
		regions:  map[string]region{},
		treepath: env.tree.Path,
	}

	extend := getExtendNode(env.tree)
	if extend == nil {
		err = s.execute(wr, env.tree.Nodes)
	} else {
		if extend.Tree == nil {
			return errors.New("template/exec: extend node is not expanded")
		}
		err = s.execute(nil, env.tree.Nodes)
		if err != nil {
			return err
		}
		s.path = extend.Path
		s.vars = []scope{builtins, globals, {}}
		err = s.execute(wr, extend.Tree.Nodes)
	}

	return err
}

// varsToScope converte delle variabili in uno scope.
func varsToScope(vars interface{}, version string) (scope, error) {

	if vars == nil {
		return scope{}, nil
	}

	var rv reflect.Value
	if rv, ok := vars.(reflect.Value); ok {
		vars = rv.Interface()
	}

	if v, ok := vars.(map[string]interface{}); ok {
		return scope(v), nil
	}

	if !rv.IsValid() {
		rv = reflect.ValueOf(vars)
	}
	rt := rv.Type()

	switch rv.Kind() {
	case reflect.Map:
		if !rt.ConvertibleTo(scopeType) {
			return nil, fmt.Errorf("template/exec: unsupported vars type")
		}
		m := rv.Convert(scopeType).Interface()
		return m.(scope), nil
	case reflect.Struct:
		globals := scope{}
		nf := rv.NumField()
		for i := 0; i < nf; i++ {
			field := rt.Field(i)
			value := rv.Field(i).Interface()
			var name string
			var ver string
			if tag, ok := field.Tag.Lookup("template"); ok {
				name, ver = parseVarTag(tag)
				if name == "" {
					return nil, fmt.Errorf("template/exec: invalid tag of field %q", field.Name)
				}
				if ver != "" && ver != version {
					continue
				}
			}
			if name == "" {
				name = field.Name
			}
			globals[name] = value
		}
		return globals, nil
	default:
		return nil, errors.New("template/exec: unsupported vars type")
	}

}

// parseVarTag esegue il parsing del tag di un campo di una struct che funge
// da variabile. Ne ritorna il nome e la versione.
func parseVarTag(tag string) (string, string) {
	sp := strings.SplitN(tag, ",", 2)
	if len(sp) == 0 {
		return "", ""
	}
	name := sp[0]
	if name == "" {
		return "", ""
	}
	for _, r := range name {
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return "", ""
		}
	}
	var version string
	if len(sp) == 2 {
		version = sp[1]
		if version == "" {
			return "", ""
		}
	}
	return name, version
}

// region definisce una region con le variabili definite nel suo scope
type region struct {
	node *ast.Region
	vars []scope
}

// state rappresenta lo stato di esecuzione di un albero.
type state struct {
	path     string
	vars     []scope
	regions  map[string]region
	treepath string
}

// errorf costruisce e ritorna un errore di esecuzione.
func (s *state) errorf(node ast.Node, format string, args ...interface{}) error {
	var pos = node.Pos()
	var err = &Error{
		Path: s.path,
		Pos: ast.Position{
			Line:   pos.Line,
			Column: pos.Column,
			Start:  pos.Start,
			End:    pos.End,
		},
		Err: fmt.Errorf(format, args...),
	}
	return err
}

// execute esegue i nodi nodes.
func (s *state) execute(wr io.Writer, nodes []ast.Node) error {

	var err error

	for _, n := range nodes {

		switch node := n.(type) {

		case *ast.Text:

			if wr != nil {
				_, err = io.WriteString(wr, node.Text[node.Cut.Left:node.Cut.Right])
				if err != nil {
					return err
				}
			}

		case *ast.Value:

			var expr interface{}
			expr, err = s.eval(node.Expr)
			if err != nil {
				return err
			}

			var str string
			switch node.Context {
			case ast.ContextHTML:
				if e, ok := expr.(io.WriterTo); ok {
					_, err = e.WriteTo(wr)
					if err != nil {
						return err
					}
				} else {
					str = interfaceToHTML(expr)
				}
			case ast.ContextScript:
				str = interfaceToScript(expr)
			}

			_, err := io.WriteString(wr, str)
			if err != nil {
				return err
			}

		case *ast.If:

			if len(node.Then) == 0 && len(node.Else) == 0 {
				continue
			}
			var expr interface{}
			expr, err = s.eval(node.Expr)
			if err != nil {
				return err
			}
			c, ok := expr.(bool)
			if !ok {
				return s.errorf(node, "non-bool %s (type %T) used as if condition", node.Expr, node.Expr)
			}
			s.vars = append(s.vars, nil)
			if c {
				if len(node.Then) > 0 {
					err = s.execute(wr, node.Then)
				}
			} else {
				if len(node.Else) > 0 {
					err = s.execute(wr, node.Else)
				}
			}
			s.vars = s.vars[:len(s.vars)-1]
			if err != nil {
				return err
			}

		case *ast.For:

			if len(node.Nodes) == 0 {
				continue
			}
			index := ""
			if node.Index != nil {
				index = node.Index.Name
			}
			ident := ""
			if node.Ident != nil {
				ident = node.Ident.Name
			}

			var expr interface{}
			expr, err = s.eval(node.Expr1)
			if err != nil {
				return err
			}

			if node.Expr2 == nil {
				// syntax: for ident in expr
				// syntax: for index, ident in expr

				setScope := func(i int, v interface{}) {
					vars := scope{ident: v}
					if index != "" {
						vars[index] = i
					}
					s.vars[len(s.vars)-1] = vars
				}

				switch vv := expr.(type) {
				case string:
					size := len(vv)
					if size == 0 {
						continue
					}
					s.vars = append(s.vars, nil)
					i := 0
					for _, v := range vv {
						setScope(i, string(v))
						err = s.execute(wr, node.Nodes)
						if err != nil {
							if err == errBreak {
								break
							}
							if err != errContinue {
								return err
							}
						}
						i++
					}
					s.vars = s.vars[:len(s.vars)-1]
				case []string:
					size := len(vv)
					if size == 0 {
						continue
					}
					s.vars = append(s.vars, nil)
					for i := 0; i < size; i++ {
						setScope(i, vv[i])
						err = s.execute(wr, node.Nodes)
						if err != nil {
							if err == errBreak {
								break
							}
							if err != errContinue {
								return err
							}
						}
					}
					s.vars = s.vars[:len(s.vars)-1]
				default:
					av := reflect.ValueOf(expr)
					if !av.IsValid() || av.Kind() != reflect.Slice {
						continue
					}
					size := av.Len()
					if size == 0 {
						continue
					}
					s.vars = append(s.vars, nil)
					for i := 0; i < size; i++ {
						setScope(i, av.Index(i).Interface())
						err = s.execute(wr, node.Nodes)
						if err != nil {
							if err == errBreak {
								break
							}
							if err != errContinue {
								return err
							}
						}
					}
					s.vars = s.vars[:len(s.vars)-1]
				}

			} else {
				// syntax: for index in expr..expr

				expr2, err := s.eval(node.Expr2)
				if err != nil {
					return err
				}

				var n1, n2 int
				{
					nn1, ok := expr.(types.Number)
					if ok {
						n1, ok = nn1.Int()
					}
					if !ok {
						return s.errorf(node, "left range value is not a integer")
					}
					nn2, ok := expr2.(types.Number)
					if ok {
						n2, ok = nn2.Int()
					}
					if !ok {
						return s.errorf(node, "right range value is not a integer")
					}
				}

				step := 1
				if n2 < n1 {
					step = -1
				}

				s.vars = append(s.vars, nil)
				for i := n1; ; i += step {
					s.vars[len(s.vars)-1] = scope{index: i}
					err = s.execute(wr, node.Nodes)
					if err != nil {
						if err == errBreak {
							break
						}
						if err == errContinue {
							continue
						}
						return err
					}
					if i == n2 {
						break
					}
				}
				s.vars = s.vars[:len(s.vars)-1]

			}

		case *ast.Break:
			return errBreak

		case *ast.Continue:
			return errContinue

		case *ast.Region:
			if wr != nil {
				return s.errorf(node.Ident, "regions not allowed", node.Pos())
			}
			var name = node.Ident.Name
			if region, ok := s.regions[name]; ok {
				return s.errorf(node.Ident, "region %s redeclared in this block\n\tprevious declaration at ",
					name, region.node.Pos())
			}
			s.regions[name] = region{
				node: node,
				vars: s.vars[0:3],
			}

		case *ast.Var:

			if s.vars[len(s.vars)-1] == nil {
				s.vars[len(s.vars)-1] = scope{}
			}
			var vars = s.vars[len(s.vars)-1]
			var name = node.Ident.Name
			if _, ok := vars[name]; ok {
				return s.errorf(node.Ident, "var %s redeclared in this block", name)
			}
			vars[name], err = s.eval(node.Expr)
			if err != nil {
				return err
			}

		case *ast.Assignment:

			var found bool
			var name = node.Ident.Name
			for i := len(s.vars) - 1; i >= 0; i-- {
				vars := s.vars[i]
				if vars != nil {
					if _, ok := vars[name]; ok {
						if i < 2 {
							if i == 0 && name == "len" {
								return s.errorf(node, "use of builtin len not in function call")
							}
							return fmt.Errorf("cannot assign to %s", name)
						}
						vars[name], err = s.eval(node.Expr)
						if err != nil {
							return err
						}
						found = true
						break
					}
				}
			}
			if !found {
				return s.errorf(node, "variable %s not declared", name)
			}

		case *ast.Show:

			name := node.Ident.Name
			reg, ok := s.regions[name]
			if !ok {
				return s.errorf(node, "undefined: %s", name)
			}
			haveSize := len(node.Arguments)
			wantSize := len(reg.node.Parameters)
			if haveSize != wantSize {
				have := "("
				for i := 0; i < haveSize; i++ {
					if i > 0 {
						have += ","
					}
					if i < wantSize {
						have += reg.node.Parameters[i].Name
					} else {
						have += "?"
					}
				}
				have += ")"
				want := "("
				for i, p := range reg.node.Parameters {
					if i > 0 {
						want += ","
					}
					want += p.Name
				}
				want += ")"
				if haveSize < wantSize {
					return s.errorf(node, "not enough arguments in show of %s\n\thave %s\n\twant %s", name, have, want)
				}
				return s.errorf(node, "too many arguments in show of %s\n\thave %s\n\twant %s", name, have, want)
			}
			var arguments = scope{}
			for i, argument := range node.Arguments {
				arguments[reg.node.Parameters[i].Name], err = s.eval(argument)
				if err != nil {
					return err
				}
			}
			vars := s.vars
			path := s.path
			s.path = s.treepath
			s.vars = append(reg.vars, arguments)
			err = s.execute(wr, reg.node.Body)
			if err != nil {
				return err
			}
			s.vars = vars
			s.path = path

		case *ast.Include:

			if node.Tree == nil {
				return s.errorf(node, "include node is not expanded")
			}
			path := s.path
			s.path = node.Path
			err = s.execute(wr, node.Tree.Nodes)
			if err != nil {
				return err
			}
			s.path = path

		}
	}

	return nil
}

// getExtendNode ritorna il nodo Extend di un albero.
// Se il nodo non è presente ritorna nil.
func getExtendNode(tree *ast.Tree) *ast.Extend {
	if len(tree.Nodes) == 0 {
		return nil
	}
	if node, ok := tree.Nodes[0].(*ast.Extend); ok {
		return node
	}
	if len(tree.Nodes) > 1 {
		if node, ok := tree.Nodes[1].(*ast.Extend); ok {
			return node
		}
	}
	return nil
}
