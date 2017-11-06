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

	globals, err := convertVars(vars, env.version)
	if err != nil {
		return err
	}

	s := state{
		vars:     []map[string]interface{}{builtins, globals, {}},
		treepath: env.tree.Path,
	}

	extend := getExtendNode(env.tree)
	if extend == nil {
		s.path = env.tree.Path
		err = s.execute(wr, env.tree.Nodes, nil)
	} else {
		if extend.Tree == nil {
			return errors.New("template/exec: extend node is not expanded")
		}
		// legge le region
		regions := map[string]*ast.Region{}
		for _, node := range env.tree.Nodes {
			if r, ok := node.(*ast.Region); ok {
				regions[r.Name] = r
			}
		}
		s.path = extend.Path
		err = s.execute(wr, extend.Tree.Nodes, regions)
	}

	return err
}

var mapStringToInterfaceType = reflect.TypeOf(map[string]interface{}{})

// convertVars converte vars in un map[string]interface{}.
func convertVars(vars interface{}, version string) (map[string]interface{}, error) {

	if vars == nil {
		return map[string]interface{}{}, nil
	}

	var rv reflect.Value
	if rv, ok := vars.(reflect.Value); ok {
		vars = rv.Interface()
	}

	if v, ok := vars.(map[string]interface{}); ok {
		return v, nil
	}

	if !rv.IsValid() {
		rv = reflect.ValueOf(vars)
	}
	rt := rv.Type()

	switch rv.Kind() {
	case reflect.Map:
		if !rt.ConvertibleTo(mapStringToInterfaceType) {
			return nil, fmt.Errorf("template/exec: unsupported vars type")
		}
		m := rv.Convert(mapStringToInterfaceType).Interface()
		return m.(map[string]interface{}), nil
	case reflect.Struct:
		globals := map[string]interface{}{}
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

// state rappresenta lo stato di esecuzione di un albero.
type state struct {
	path     string
	vars     []map[string]interface{}
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
func (s *state) execute(wr io.Writer, nodes []ast.Node, regions map[string]*ast.Region) error {

	var err error

	for _, n := range nodes {

		switch node := n.(type) {

		case *ast.Text:

			_, err = io.WriteString(wr, node.Text[node.Cut.Left:node.Cut.Right])
			if err != nil {
				return err
			}

		case *ast.Show:

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
			if c, ok := expr.(bool); ok {
				if c {
					if len(node.Then) > 0 {
						err = s.execute(wr, node.Then, nil)
					}
				} else {
					if len(node.Else) > 0 {
						err = s.execute(wr, node.Else, nil)
					}
				}
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("non-bool %s (type %T) used as if condition", node.Expr, node.Expr)
			}

		case *ast.For:

			if len(node.Nodes) == 0 {
				continue
			}
			index := ""
			if node.Index != nil {
				index = node.Index.Name
			}
			ident := node.Ident.Name

			var expr interface{}
			expr, err = s.eval(node.Expr)
			if err != nil {
				return err
			}

			av := reflect.ValueOf(expr)
			if !av.IsValid() {
				continue
			}

			var list []interface{}
			if av.Kind() == reflect.Slice {
				if av.Len() == 0 {
					continue
				}
				list = make([]interface{}, av.Len())
				for i := 0; i < len(list); i++ {
					list[i] = av.Index(i).Interface()
				}
			} else {
				list = []interface{}{av.Interface()}
			}

			s.vars = append(s.vars, nil)
			for i, v := range list {
				vars := map[string]interface{}{ident: v}
				if index != "" {
					vars[index] = i
				}
				s.vars[len(s.vars)-1] = vars
				err = s.execute(wr, node.Nodes, nil)
				if err != nil {
					if err == errBreak {
						break
					}
					if err == errContinue {
						continue
					}
					return err
				}
			}
			s.vars = s.vars[:len(s.vars)-1]

		case *ast.Break:
			return errBreak

		case *ast.Continue:
			return errContinue

		case *ast.Var:

			var vars = s.vars[len(s.vars)-1]
			var name = node.Ident.Name
			if _, ok := vars[name]; ok {
				return fmt.Errorf("variable %q already declared in this block: %#v", name, vars)
			}
			vars[name], err = s.eval(node.Expr)
			if err != nil {
				return err
			}

		case *ast.Assignment:

			var name = node.Ident.Name
			for i := len(s.vars) - 1; i >= 0; i-- {
				var vars = s.vars[i]
				if _, ok := vars[name]; ok {
					if i < 2 {
						if i == 0 && name == "len" {
							return fmt.Errorf("use of builtin len not in function call")
						}
						return fmt.Errorf("cannot assign to %s", name)
					}
					vars[name], err = s.eval(node.Expr)
					if err != nil {
						return err
					}
					break
				}
			}
			return fmt.Errorf("variable %s not declared", name)

		case *ast.Region:

			if regions != nil {
				region := regions[node.Name]
				if region != nil {
					path := s.path
					s.path = s.treepath
					err = s.execute(wr, region.Nodes, nil)
					if err != nil {
						return err
					}
					s.path = path
				}
			}

		case *ast.Include:

			if node.Tree == nil {
				return errors.New("include node is not expanded")
			}
			path := s.path
			s.path = node.Path
			err = s.execute(wr, node.Tree.Nodes, nil)
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
