// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"open2b/template/ast"
	"open2b/template/ast/astutil"

	"github.com/shopspring/decimal"
)

var zero = decimal.New(0, 0)
var decimalType = reflect.TypeOf(zero)

const maxInt = int64(^uint(0) >> 1)
const minInt = -int64(^uint(0)>>1) - 1

// Slice implements the slice values.
type Slice []interface{}

// eval evaluates expr in a single-value context and returns its value.
func (r *rendering) eval(exp ast.Expression) (value interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				panic(r)
			}
		}
	}()
	return r.evalExpression(exp), nil
}

// eval0 evaluates expr in a void context. It returns an error if the
// expression evaluates to a value.
func (r *rendering) eval0(expr ast.Expression) error {
	if e, ok := expr.(*ast.Call); ok {
		_, err := r.evalCallN(e, 0)
		return err
	}
	_, err := r.eval(expr)
	if err != nil {
		return err
	}
	return r.errorf(expr, "%s evaluated but not used", expr)
}

// eval2 evaluates expr in a two-value context and returns its values.
func (r *rendering) eval2(expr ast.Expression) (v1, v2 interface{}, err error) {
	switch e := expr.(type) {
	case *ast.TypeAssertion:
		v, err := r.eval(e.Expr)
		if err != nil {
			return nil, false, err
		}
		var typ valuetype
		for i := len(r.vars) - 1; i >= 0; i-- {
			if r.vars[i] != nil {
				if i == 0 && e.Type.Name == "len" {
					return nil, false, r.errorf(e, "use of builtin len not in function call")
				}
				if v, ok := r.vars[i][e.Type.Name]; ok {
					if vt, ok := v.(valuetype); ok {
						typ = vt
						break
					}
					return nil, false, r.errorf(e.Type, "%s is not a type", e.Type.Name)
				}
			}
			if i == 0 {
				return nil, false, r.errorf(e.Type, "undefined: %s", e.Type.Name)
			}
		}
		if !hasType(v, typ) {
			return nil, false, nil
		}
		return v, true, nil
	case *ast.Selector:
		return r.evalSelector2(e)
	case *ast.Index:
		return r.evalIndex2(e, 2)
	case *ast.Call:
		values, err := r.evalCallN(e, 2)
		if err != nil {
			return nil, nil, err
		}
		return values[0].Interface(), values[1].Interface(), nil
	case *ast.Identifier:
		for i := len(r.vars) - 1; i >= 0; i-- {
			if r.vars[i] != nil {
				if i == 0 && e.Name == "len" {
					return nil, false, r.errorf(e, "use of builtin len not in function call")
				}
				if v, ok := r.vars[i][e.Name]; ok {
					return v, true, nil
				}
			}
		}
		return nil, false, nil
	}
	return nil, nil, r.errorf(expr, "assignment mismatch: 2 variables but 1 values")
}

// evalN evaluates expr in a n-value context, with n > 2, and returns its
// values.
func (r *rendering) evalN(expr ast.Expression, n int) ([]reflect.Value, error) {
	if e, ok := expr.(*ast.Call); ok {
		return r.evalCallN(e, n)
	}
	return nil, r.errorf(expr, "assignment mismatch: %d variables but 1 values", n)
}

// evalExpression evaluates an expression and returns its value.
// In the event of an error, calls panic with the error as parameter.
func (r *rendering) evalExpression(expr ast.Expression) interface{} {
	switch e := expr.(type) {
	case *ast.String:
		return e.Text
	case *ast.Int:
		return e.Value
	case *ast.Number:
		return e.Value
	case *ast.Parentesis:
		return r.evalExpression(e.Expr)
	case *ast.UnaryOperator:
		return r.evalUnaryOperator(e)
	case *ast.BinaryOperator:
		return r.evalBinaryOperator(e)
	case *ast.Identifier:
		return r.evalIdentifier(e)
	case *ast.Map:
		return r.evalMap(e)
	case *ast.Slice:
		return r.evalSlice(e)
	case *ast.Call:
		return r.evalCall(e)
	case *ast.Index:
		return r.evalIndex(e)
	case *ast.Slicing:
		return r.evalSlicing(e)
	case *ast.Selector:
		return r.evalSelector(e)
	case *ast.TypeAssertion:
		return r.evalTypeAssertion(e)
	default:
		panic(r.errorf(expr, "unexpected node type %s", typeof(expr)))
	}
}

// evalUnaryOperator evaluates a unary operator and returns its value.
// On error it calls panic with the error as argument.
func (r *rendering) evalUnaryOperator(node *ast.UnaryOperator) interface{} {
	var expr = asBase(r.evalExpression(node.Expr))
	switch node.Op {
	case ast.OperatorNot:
		if b, ok := expr.(bool); ok {
			return !b
		}
		panic(r.errorf(node, "invalid operation: ! %s", typeof(expr)))
	case ast.OperatorAddition:
		switch n := expr.(type) {
		case decimal.Decimal:
			return n
		case int:
			return n
		}
		panic(r.errorf(node, "invalid operation: + %s", typeof(expr)))
	case ast.OperatorSubtraction:
		switch n := expr.(type) {
		case decimal.Decimal:
			return n.Neg()
		case int:
			if n == int(minInt) {
				return decimal.New(int64(n), 0).Neg()
			}
			return -n
		}
		panic(r.errorf(node, "invalid operation: - %s", typeof(expr)))
	}
	panic("unknown unary operator")
}

// evalBinaryOperator evaluates a binary operator and returns its value.
// On error it calls panic with the error as argument.
func (r *rendering) evalBinaryOperator(op *ast.BinaryOperator) interface{} {

	switch op.Op {

	case ast.OperatorEqual:
		return r.evalEqual(op)

	case ast.OperatorNotEqual:
		return !r.evalEqual(op)

	case ast.OperatorLess:
		return r.evalLess(op)

	case ast.OperatorLessOrEqual:
		return !r.evalGreater(op)

	case ast.OperatorGreater:
		return r.evalGreater(op)

	case ast.OperatorGreaterOrEqual:
		return !r.evalLess(op)

	case ast.OperatorAnd:
		expr1 := asBase(r.evalExpression(op.Expr1))
		if e1, ok := expr1.(bool); ok {
			if !e1 {
				return false
			}
			expr2 := asBase(r.evalExpression(op.Expr2))
			if e2, ok := expr2.(bool); ok {
				return e1 && e2
			}
			panic(r.errorf(op, "invalid operation: %s && %s", typeof(expr1), typeof(expr2)))
		}
		panic(r.errorf(op, "invalid operation: %s && ...", typeof(expr1)))

	case ast.OperatorOr:
		expr1 := asBase(r.evalExpression(op.Expr1))
		if e1, ok := expr1.(bool); ok {
			if e1 {
				return true
			}
			expr2 := asBase(r.evalExpression(op.Expr2))
			if e2, ok := expr2.(bool); ok {
				return e1 || e2
			}
			panic(r.errorf(op, "invalid operation: %s || %s", typeof(expr1), typeof(expr2)))
		}
		panic(r.errorf(op, "invalid operation: %s || ...", typeof(expr1)))

	case ast.OperatorAddition:
		expr1 := asBase(r.evalExpression(op.Expr1))
		expr2 := asBase(r.evalExpression(op.Expr2))
		switch e1 := expr1.(type) {
		case string:
			switch e2 := expr2.(type) {
			case string:
				return e1 + e2
			case HTML:
				return HTML(htmlEscapeString(e1) + string(e2))
			}
		case HTML:
			switch e2 := expr2.(type) {
			case string:
				return HTML(string(e1) + htmlEscapeString(e2))
			case HTML:
				return HTML(string(e1) + string(e2))
			}
		case decimal.Decimal:
			switch e2 := expr2.(type) {
			case decimal.Decimal:
				return e1.Add(e2)
			case int:
				return e1.Add(decimal.New(int64(e2), 0))
			}
		case int:
			switch e2 := expr2.(type) {
			case decimal.Decimal:
				return decimal.New(int64(e1), 0).Add(e2)
			case int:
				e := e1 + e2
				if (e < e1) != (e2 < 0) {
					return decimal.New(int64(e1), 0).Add(decimal.New(int64(e2), 0))
				}
				return e
			}
		}
		panic(r.errorf(op, "invalid operation: %s + %s", typeof(expr1), typeof(expr2)))

	case ast.OperatorSubtraction:
		expr1 := asBase(r.evalExpression(op.Expr1))
		expr2 := asBase(r.evalExpression(op.Expr2))
		switch e1 := expr1.(type) {
		case decimal.Decimal:
			switch e2 := expr2.(type) {
			case decimal.Decimal:
				return e1.Sub(e2)
			case int:
				return e1.Sub(decimal.New(int64(e2), 0))
			}
		case int:
			switch e2 := expr2.(type) {
			case decimal.Decimal:
				return decimal.New(int64(e1), 0).Sub(e2)
			case int:
				e := e1 - e2
				if (e < e1) != (e2 > 0) {
					return decimal.New(int64(e1), 0).Sub(decimal.New(int64(e2), 0))
				}
				return e
			}
		}
		panic(r.errorf(op, "invalid operation: %s - %s", typeof(expr1), typeof(expr2)))

	case ast.OperatorMultiplication:
		expr1 := asBase(r.evalExpression(op.Expr1))
		expr2 := asBase(r.evalExpression(op.Expr2))
		switch e1 := expr1.(type) {
		case decimal.Decimal:
			switch e2 := expr2.(type) {
			case decimal.Decimal:
				return e1.Mul(e2)
			case int:
				return e1.Mul(decimal.New(int64(e2), 0))
			}
		case int:
			switch e2 := expr2.(type) {
			case decimal.Decimal:
				return decimal.New(int64(e1), 0).Mul(e2)
			case int:
				if e1 == 0 || e2 == 0 {
					return 0
				}
				e := e1 * e2
				if (e < 0) != ((e1 < 0) != (e2 < 0)) || e/e2 != e1 {
					return decimal.New(int64(e1), 0).Mul(decimal.New(int64(e2), 0))
				}
				return e
			}
		}
		panic(r.errorf(op, "invalid operation: %s * %s", typeof(expr1), typeof(expr2)))

	case ast.OperatorDivision:
		expr1 := asBase(r.evalExpression(op.Expr1))
		expr2 := asBase(r.evalExpression(op.Expr2))
		switch e2 := expr2.(type) {
		case decimal.Decimal:
			if e2.IsZero() {
				panic(r.errorf(op, "number divide by zero"))
			}
			switch e1 := expr1.(type) {
			case decimal.Decimal:
				return e1.DivRound(e2, 20)
			case int:
				return decimal.New(int64(e1), 0).DivRound(e2, 20)
			}
		case int:
			if e2 == 0 {
				panic(r.errorf(op, "number divide by zero"))
			}
			switch e1 := expr1.(type) {
			case decimal.Decimal:
				return e1.DivRound(decimal.New(int64(e2), 0), 20)
			case int:
				if e1%e2 == 0 && !(e1 == int(minInt) && e2 == -1) {
					return e1 / e2
				}
				return decimal.New(int64(e1), 0).DivRound(decimal.New(int64(e2), 0), 20)
			}
		}
		panic(r.errorf(op, "invalid operation: %s / %s", typeof(expr1), typeof(expr2)))

	case ast.OperatorModulo:
		expr1 := asBase(r.evalExpression(op.Expr1))
		expr2 := asBase(r.evalExpression(op.Expr2))
		switch e1 := expr1.(type) {
		case decimal.Decimal:
			switch e2 := expr2.(type) {
			case decimal.Decimal:
				return e1.Mod(e2)
			case int:
				return e1.Mod(decimal.New(int64(e2), 0))
			}
		case int:
			switch e2 := expr2.(type) {
			case decimal.Decimal:
				return decimal.New(int64(e1), 0).Mod(e2)
			case int:
				return e1 % e2
			}
		}
		panic(r.errorf(op, "invalid operation: %s %% %s", typeof(expr1), typeof(expr2)))

	}

	panic("unknown binary operator")
}

// evalEqual evaluates op as a binary equal operator and returns its value.
// On error it calls panic with the error as argument.
func (r *rendering) evalEqual(op *ast.BinaryOperator) bool {

	expr1 := asBase(r.evalExpression(op.Expr1))
	expr2 := asBase(r.evalExpression(op.Expr2))

	switch e1 := expr1.(type) {
	case string:
		switch e2 := expr2.(type) {
		case string:
			return e1 == e2
		case HTML:
			return e1 == string(e2)
		}
	case HTML:
		switch e2 := expr2.(type) {
		case string:
			return string(e1) == e2
		case HTML:
			return e1 == e2
		}
	case decimal.Decimal:
		switch e2 := expr2.(type) {
		case decimal.Decimal:
			return e1.Equal(e2)
		case int:
			return e1.Cmp(decimal.New(int64(e2), 0)) == 0
		}
	case int:
		switch e2 := expr2.(type) {
		case decimal.Decimal:
			return decimal.New(int64(e1), 0).Cmp(e2) == 0
		case int:
			return e1 == e2
		}
	case bool:
		if e2, ok := expr2.(bool); ok {
			return e1 == e2
		}
	default:
		uNil1 := expr1 == nil && r.isBuiltin("nil", op.Expr1)
		uNil2 := expr2 == nil && r.isBuiltin("nil", op.Expr2)
		if uNil1 && uNil2 {
			panic(r.errorf(op, "invalid operation: nil %s nil", op.Op))
		}
		if uNil2 && (expr1 == nil || reflect.ValueOf(expr1).IsNil()) {
			return true
		}
		if uNil1 && (expr2 == nil || reflect.ValueOf(expr2).IsNil()) {
			return true
		}
	}

	return false
}

// evalLess evaluates op as a binary less operator and returns its value.
// On error it calls panic with the error as argument.
func (r *rendering) evalLess(op *ast.BinaryOperator) bool {

	expr1 := asBase(r.evalExpression(op.Expr1))
	expr2 := asBase(r.evalExpression(op.Expr2))

	switch e1 := expr1.(type) {
	case string:
		switch e2 := expr2.(type) {
		case string:
			return e1 < e2
		case HTML:
			return e1 < string(e2)
		}
	case HTML:
		switch e2 := expr2.(type) {
		case string:
			return string(e1) < e2
		case HTML:
			return e1 < e2
		}
	case decimal.Decimal:
		switch e2 := expr2.(type) {
		case decimal.Decimal:
			return e1.Cmp(e2) < 0
		case int:
			return e1.Cmp(decimal.New(int64(e2), 0)) < 0
		}
	case int:
		switch e2 := expr2.(type) {
		case decimal.Decimal:
			return decimal.New(int64(e1), 0).Cmp(e2) < 0
		case int:
			return e1 < e2
		}
	}

	panic(r.errorf(op, "invalid operation: %s %s %s", typeof(expr1), op.Op, typeof(expr2)))
}

// evalGreater evaluates op as a binary greater operator and returns its
// value. On error it calls panic with the error as argument.
func (r *rendering) evalGreater(op *ast.BinaryOperator) bool {

	expr1 := asBase(r.evalExpression(op.Expr1))
	expr2 := asBase(r.evalExpression(op.Expr2))

	switch e1 := expr1.(type) {
	case string:
		switch e2 := expr2.(type) {
		case string:
			return e1 > e2
		case HTML:
			return e1 > string(e2)
		}
	case HTML:
		switch e2 := expr2.(type) {
		case string:
			return string(e1) > e2
		case HTML:
			return e1 > e2
		}
	case decimal.Decimal:
		switch e2 := expr2.(type) {
		case decimal.Decimal:
			return e1.Cmp(e2) > 0
		case int:
			return e1.Cmp(decimal.New(int64(e2), 0)) > 0
		}
	case int:
		switch e2 := expr2.(type) {
		case decimal.Decimal:
			return decimal.New(int64(e1), 0).Cmp(e2) > 0
		case int:
			return e1 > e2
		}
	}

	panic(r.errorf(op, "invalid operation: %s %s %s", typeof(expr1), op.Op, typeof(expr2)))
}

// evalMap evaluates a map expression and returns its value.
func (r *rendering) evalMap(node *ast.Map) interface{} {
	elements := make(Map, len(node.Elements))
	for _, element := range node.Elements {
		key := asBase(r.evalExpression(element.Key))
		switch key.(type) {
		case nil, string, HTML, decimal.Decimal, int, bool:
		default:
			panic(r.errorf(node, "hash of unhashable type %s", typeof(key)))
		}
		elements.Store(key, r.evalExpression(element.Value))
	}
	return elements
}

// evalSlice evaluates a slice expression and returns its value.
func (r *rendering) evalSlice(node *ast.Slice) interface{} {
	elements := make(Slice, len(node.Elements))
	for i, element := range node.Elements {
		elements[i] = r.evalExpression(element)
	}
	return elements
}

// evalSelector evaluates a selector expression and returns its value.
func (r *rendering) evalSelector(node *ast.Selector) interface{} {
	v, ok, err := r.evalSelector2(node)
	if err != nil {
		panic(err)
	}
	if !ok {
		panic(r.errorf(node, "field %q does not exist", node.Ident))
	}
	return v
}

// evalSelector2 evaluates a selector in a two-values context. If its
// identifier exists it returns the value and true, otherwise it returns nil
// and false.
func (r *rendering) evalSelector2(node *ast.Selector) (interface{}, bool, error) {
	value := asBase(r.evalExpression(node.Expr))
	// map
	if v2, ok := value.(map[string]interface{}); ok {
		if v3, ok := v2[node.Ident]; ok {
			return v3, true, nil
		}
		return nil, false, nil
	}
	switch v := value.(type) {
	case Map:
		vv, ok := v.Load(node.Ident)
		return vv, ok, nil
	case map[string]interface{}:
		vv, ok := v[node.Ident]
		return vv, ok, nil
	case map[string]string:
		vv, ok := v[node.Ident]
		return vv, ok, nil
	case map[string]HTML:
		vv, ok := v[node.Ident]
		return vv, ok, nil
	case map[string]decimal.Decimal:
		vv, ok := v[node.Ident]
		return vv, ok, nil
	case map[string]int:
		vv, ok := v[node.Ident]
		return vv, ok, nil
	case map[string]bool:
		vv, ok := v[node.Ident]
		return vv, ok, nil
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Map:
		if rv.Type().Key().Kind() == reflect.String {
			v := rv.MapIndex(reflect.ValueOf(node.Ident))
			if !v.IsValid() {
				return nil, false, nil
			}
			return v.Interface(), true, nil
		}
		return nil, false, r.errorf(node, "unsupported vars type")
	case reflect.Struct, reflect.Ptr:
		keys := structKeys(rv)
		if keys == nil {
			return nil, false, r.errorf(node, "unsupported vars type")
		}
		sk, ok := keys[node.Ident]
		if !ok {
			return nil, false, nil
		}
		return sk.value(rv), true, nil
	}
	return nil, false, r.errorf(node, "invalid operation: %s (type %s is not map)", node, typeof(value))
}

// evalTypeAssertion evaluates a type assertion.
func (r *rendering) evalTypeAssertion(node *ast.TypeAssertion) interface{} {
	val := r.evalExpression(node.Expr)
	ide := r.evalIdentifier(node.Type)
	typ, ok := ide.(valuetype)
	if !ok {
		panic(r.errorf(node.Type, "%s is not a type", node.Type.Name))
	}
	if !hasType(val, typ) {
		panic(r.errorf(node.Type, "%s is %s, not %s", node.Expr, typeof(val), typ))
	}
	return val
}

// hasType indicates if v has type typ.
func hasType(v interface{}, typ valuetype) bool {
	if v == nil {
		return false
	}
	v = asBase(v)
	switch vv := v.(type) {
	case string:
		return typ == builtins["string"]
	case HTML:
		return typ == builtins["string"] || typ == builtins["html"]
	case decimal.Decimal:
		if typ == builtins["number"] {
			return true
		}
		if typ != builtins["int"] {
			return false
		}
		if vv.LessThan(minIntAsDecimal) || maxIntAsDecimal.LessThan(vv) {
			return false
		}
		p := vv.IntPart()
		return decimal.New(p, 0).Equal(vv)
	case int:
		return typ == builtins["int"] || typ == builtins["number"]
	case bool:
		return typ == builtins["bool"]
	case Slice, []interface{}, []string, []HTML, []decimal.Decimal, []int, []bool:
		return typ == builtins["slice"]
	case Map, map[string]interface{}, map[string]string, map[string]HTML,
		map[string]decimal.Decimal, map[string]int, map[string]bool:
		return typ == builtins["map"]
	case error:
		return typ == builtins["error"]
	}
	switch typ {
	case builtins["string"], builtins["html"], builtins["number"], builtins["int"], builtins["bool"], builtins["error"]:
		return false
	}
	switch rv := reflect.ValueOf(v); rv.Kind() {
	case reflect.Struct, reflect.Map:
		return typ == builtins["map"]
	case reflect.Ptr:
		return typ == builtins["map"] && reflect.Indirect(rv).Kind() == reflect.Struct
	case reflect.Slice:
		return typ == builtins["slice"]
	}
	return false
}

// evalIndex evaluates an index expression in a single context.
func (r *rendering) evalIndex(node *ast.Index) interface{} {
	v, _, err := r.evalIndex2(node, 1)
	if err != nil {
		panic(err)
	}
	return v
}

// evalIndex2 evaluates an index expression in a single ( n == 1 ) or
// two-values ( n == 2 ) context. If its index exists, returns the value and
// true, otherwise it returns nil and false.
func (r *rendering) evalIndex2(node *ast.Index, n int) (interface{}, bool, error) {
	v, err := r.eval(node.Expr)
	if err != nil {
		return nil, false, err
	}
	if v == nil {
		if r.isBuiltin("nil", node.Expr) {
			return nil, false, r.errorf(node.Expr, "use of untyped nil")
		} else {
			return nil, false, r.errorf(node, "index out of range")
		}
	}
	checkSlice := func(length int) (int, error) {
		if n == 2 {
			return 0, r.errorf(node, "assignment mismatch: 2 variables but 1 values")
		}
		i, err := r.sliceIndex(node.Index)
		if err != nil {
			return 0, err
		}
		if i >= length {
			return 0, r.errorf(node, "index out of range")
		}
		return i, nil
	}
	v = asBase(v)
	if vv, ok := v.(HTML); ok {
		v = string(vv)
	}
	switch vv := v.(type) {
	case string:
		if n == 2 {
			return nil, false, r.errorf(node, "assignment mismatch: 2 variables but 1 values")
		}
		i, err := r.sliceIndex(node.Index)
		if err != nil {
			return nil, false, err
		}
		if i >= len(vv) {
			return nil, false, r.errorf(node, "index out of range")
		}
		p := 0
		for _, c := range vv {
			if p == i {
				return string(c), true, nil
			}
			p++
		}
		return nil, false, r.errorf(node, "index out of range")
	case Map:
		u, ok := vv.Load(asBase(r.evalExpression(node.Index)))
		return u, ok, nil
	case map[string]interface{}:
		k := asBase(r.evalExpression(node.Index))
		if s, ok := k.(string); ok {
			if u, ok := vv[s]; ok {
				return u, true, nil
			}
		}
		return nil, false, nil
	case map[string]string:
		k := asBase(r.evalExpression(node.Index))
		if s, ok := k.(string); ok {
			if u, ok := vv[s]; ok {
				return u, true, nil
			}
		}
		return nil, false, nil
	case map[string]HTML:
		k := asBase(r.evalExpression(node.Index))
		if s, ok := k.(string); ok {
			if u, ok := vv[s]; ok {
				return u, true, nil
			}
		}
		return nil, false, nil
	case map[string]decimal.Decimal:
		k := asBase(r.evalExpression(node.Index))
		if s, ok := k.(string); ok {
			if u, ok := vv[s]; ok {
				return u, true, nil
			}
		}
		return nil, false, nil
	case map[string]int:
		k := asBase(r.evalExpression(node.Index))
		if s, ok := k.(string); ok {
			if u, ok := vv[s]; ok {
				return u, true, nil
			}
		}
		return nil, false, nil
	case map[string]bool:
		k := asBase(r.evalExpression(node.Index))
		if s, ok := k.(string); ok {
			if u, ok := vv[s]; ok {
				return u, true, nil
			}
		}
		return nil, false, nil
	case Slice:
		i, err := checkSlice(len(vv))
		if err != nil {
			return nil, false, err
		}
		return vv[i], true, nil
	case []interface{}:
		i, err := checkSlice(len(vv))
		if err != nil {
			return nil, false, err
		}
		return vv[i], true, nil
	case []string:
		i, err := checkSlice(len(vv))
		if err != nil {
			return nil, false, err
		}
		return vv[i], true, nil
	case []HTML:
		i, err := checkSlice(len(vv))
		if err != nil {
			return nil, false, err
		}
		return vv[i], true, nil
	case []decimal.Decimal:
		i, err := checkSlice(len(vv))
		if err != nil {
			return nil, false, err
		}
		return vv[i], true, nil
	case []int:
		i, err := checkSlice(len(vv))
		if err != nil {
			return nil, false, err
		}
		return vv[i], true, nil
	case []bool:
		i, err := checkSlice(len(vv))
		if err != nil {
			return nil, false, err
		}
		return vv[i], true, nil
	}
	var rv = reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Map:
		k := asBase(r.evalExpression(node.Index))
		val := reflect.ValueOf(k)
		if val.IsValid() && val.Type().AssignableTo(rv.Type().Key()) {
			return val.Interface(), true, nil
		}
		return nil, false, nil
	case reflect.Struct, reflect.Ptr:
		if keys := structKeys(rv); keys != nil {
			k := asBase(r.evalExpression(node.Index))
			if k, ok := k.(string); ok {
				if sk, ok := keys[k]; ok {
					return sk.value(rv), true, nil
				}
			}
			return nil, false, nil
		}
	case reflect.Slice:
		i, err := checkSlice(rv.Len())
		if err != nil {
			return nil, false, err
		}
		return rv.Index(i).Interface(), true, nil
	}
	return nil, false, r.errorf(node, "invalid operation: %s (type %s does not support indexing)", node, typeof(v))
}

// sliceIndex evaluates node as a slice index and returns the value.
func (r *rendering) sliceIndex(node ast.Expression) (int, error) {
	var i int
	switch index := asBase(r.evalExpression(node)).(type) {
	case int:
		i = index
	case decimal.Decimal:
		var err error
		i, err = r.decimalToInt(node, index)
		if err != nil {
			return 0, err
		}
		if i < 0 {
			return 0, r.errorf(node, "invalid slice index %s (index must be non-negative)", node)
		}
	default:
		return 0, r.errorf(node, "non-integer slice index %s", node)
	}
	if i < 0 {
		return 0, r.errorf(node, "invalid slice index %d (index must be non-negative)", i)
	}
	return i, nil
}

// evalSlicing evaluates a slice expression and returns its value.
func (r *rendering) evalSlicing(node *ast.Slicing) interface{} {
	var ok bool
	var l, h int
	if node.Low != nil {
		n := r.evalExpression(node.Low)
		l, ok = n.(int)
		if !ok {
			panic(r.errorf(node.Low, "invalid slice index %s (type %s)", node.Low, typeof(n)))
		}
		if l < 0 {
			panic(r.errorf(node.Low, "invalid slice index %s (index must be non-negative)", node.Low))
		}
	}
	if node.High != nil {
		n := r.evalExpression(node.High)
		h, ok = n.(int)
		if !ok {
			panic(r.errorf(node.High, "invalid slice index %s (type %s)", node.High, typeof(n)))
		}
		if h < 0 {
			panic(r.errorf(node.High, "invalid slice index %s (index must be non-negative)", node.High))
		}
		if l > h {
			panic(r.errorf(node.Low, "invalid slice index: %d > %d", l, h))
		}
	}
	var v = asBase(r.evalExpression(node.Expr))
	if v == nil {
		if r.isBuiltin("nil", node.Expr) {
			panic(r.errorf(node.Expr, "use of untyped nil"))
		} else {
			panic(r.errorf(node, "slice bounds out of range"))
		}
	}
	h2 := func(length int) int {
		if node.High == nil {
			h = length
		} else if h > length {
			panic(r.errorf(node.High, "slice bounds out of range"))
		}
		return h
	}
	if vv, ok := v.(HTML); ok {
		v = string(vv)
	}
	switch vv := v.(type) {
	case string:
		i := 0
		lb, hb := -1, -1
		for ib := range vv {
			if i == l {
				lb = ib
				if node.High == nil {
					hb = len(vv)
					break
				}
			}
			if i >= l && i == h {
				hb = ib
				break
			}
			i++
		}
		if lb == -1 {
			panic(r.errorf(node.Low, "slice bounds out of range"))
		}
		if hb == -1 {
			if i < h {
				panic(r.errorf(node.High, "slice bounds out of range"))
			}
			hb = len(vv)
		}
		if lb == 0 && hb == len(vv) {
			return vv
		}
		return vv[lb:hb]
	case Slice:
		return vv[l:h2(len(vv))]
	case []interface{}:
		return vv[l:h2(len(vv))]
	case []string:
		return vv[l:h2(len(vv))]
	case []HTML:
		return vv[l:h2(len(vv))]
	case []decimal.Decimal:
		return vv[l:h2(len(vv))]
	case []int:
		return vv[l:h2(len(vv))]
	case []bool:
		return vv[l:h2(len(vv))]
	}
	if e := reflect.ValueOf(v); e.Kind() == reflect.Slice {
		return e.Slice(l, h2(e.Len())).Interface()
	}
	panic(r.errorf(node, "cannot slice %s (type %s)", node.Expr, typeof(v)))
}

// evalIdentifier evaluates an identifier expression and returns its value.
func (r *rendering) evalIdentifier(node *ast.Identifier) interface{} {
	for i := len(r.vars) - 1; i >= 0; i-- {
		if r.vars[i] != nil {
			if i == 0 && node.Name == "len" {
				panic(r.errorf(node, "use of builtin len not in function call"))
			}
			if v, ok := r.vars[i][node.Name]; ok {
				return v
			}
		}
	}
	panic(r.errorf(node, "undefined: %s", node.Name))
}

// evalCall evaluates a call expression in a single-value context. It returns
// an error if the function
func (r *rendering) evalCall(node *ast.Call) interface{} {
	results, err := r.evalCallN(node, 1)
	if err != nil {
		panic(err)
	}
	return results[0].Interface()
}

// evalCallN evaluates a call expression in n-values context and returns its
// values. It returns an error if n > 0 and the function does not return n
// values.
func (r *rendering) evalCallN(node *ast.Call, n int) ([]reflect.Value, error) {

	if r.isBuiltin("len", node.Func) {
		err := r.checkBuiltInParameterCount(node, 1, 1, n)
		if err != nil {
			return nil, err
		}
		arg := asBase(r.evalExpression(node.Args[0]))
		length := _len(arg)
		if length == -1 {
			return nil, r.errorf(node.Args[0], "invalid argument %s (type %s) for len", node.Args[0], typeof(arg))
		}
		return []reflect.Value{reflect.ValueOf(length)}, nil
	}

	if r.isBuiltin("delete", node.Func) {
		err := r.checkBuiltInParameterCount(node, 2, 0, n)
		if err != nil {
			return nil, err
		}
		arg := asBase(r.evalExpression(node.Args[0]))
		m, ok := arg.(Map)
		if !ok {
			typ := typeof(arg)
			if typ == "map" {
				return nil, r.errorf(node, "cannot delete from non-mutable map")
			} else {
				return nil, r.errorf(node, "first argument to delete must be map; have %s", typ)
			}
		}
		k := asBase(r.evalExpression(node.Args[1]))
		_delete(m, k)
		return nil, nil
	}

	var f = r.evalExpression(node.Func)

	if typ, ok := f.(valuetype); ok {
		if len(node.Args) == 0 {
			return nil, r.errorf(node, "missing argument to conversion to %s: %s", typ, node)
		}
		if len(node.Args) > 1 {
			return nil, r.errorf(node, "too many arguments to conversion to %s: %s", typ, node)
		}
		v, err := r.convert(node.Args[0], typ)
		if err != nil {
			panic(r.errorf(node.Args[0], "%s", err))
		}
		return []reflect.Value{reflect.ValueOf(v)}, nil
	}

	var fun = reflect.ValueOf(f)
	if !fun.IsValid() {
		return nil, r.errorf(node, "cannot call non-function %s (type %s)", node.Func, typeof(f))
	}
	var typ = fun.Type()
	if typ.Kind() != reflect.Func {
		return nil, r.errorf(node, "cannot call non-function %s (type %s)", node.Func, typeof(f))
	}

	var numOut = typ.NumOut()
	switch {
	case n == 1 && numOut > 1:
		expr := astutil.CloneExpression(node).(*ast.Call)
		expr.Args = nil
		return nil, r.errorf(node, "multiple-value %s in single-value context", expr)
	case n > 0 && numOut == 0:
		return nil, r.errorf(node, "%s used as value", node)
	case n > 0 && n != numOut:
		return nil, r.errorf(node, "assignment mismatch: %d variables but %d values", n, numOut)
	}

	var numIn = typ.NumIn()
	var isVariadic = typ.IsVariadic()

	if (!isVariadic && len(node.Args) != numIn) || (isVariadic && len(node.Args) < numIn-1) {
		have := "("
		for i, arg := range node.Args {
			if i > 0 {
				have += ", "
			}
			have += typeof(r.evalExpression(arg))
		}
		have += ")"
		want := "("
		for i := 0; i < numIn; i++ {
			if i > 0 {
				want += ", "
			}
			if i == numIn-1 && isVariadic {
				want += "..."
			}
			if in := typ.In(i); in.Kind() == reflect.Interface {
				want += "any"
			} else {
				want += typeof(reflect.Zero(in).Interface())
			}
		}
		want += ")"
		if len(node.Args) < numIn {
			return nil, r.errorf(node, "not enough arguments in call to %s\n\thave %s\n\twant %s", node.Func, have, want)
		} else {
			return nil, r.errorf(node, "too many arguments in call to %s\n\thave %s\n\twant %s", node.Func, have, want)
		}
	}

	var args = make([]reflect.Value, len(node.Args))

	var lastIn = numIn - 1
	var in reflect.Type

	for i := 0; i < len(node.Args); i++ {
		if i < lastIn || !isVariadic {
			in = typ.In(i)
		} else if i == lastIn {
			in = typ.In(lastIn).Elem()
		}

		inKind := in.Kind()
		var arg interface{}
		if i < len(node.Args) {
			arg = asBase(r.evalExpression(node.Args[i]))
		}
		if arg == nil {
			if i < len(node.Args) {
				if in == decimalType {
					return nil, r.errorf(node, "cannot use nil as type number in argument to %s", node.Func)
				}
				switch inKind {
				case reflect.Bool, reflect.Int, reflect.String:
					wantType := typeof(reflect.Zero(in).Interface())
					return nil, r.errorf(node, "cannot use nil as type %s in argument to %s", wantType, node.Func)
				}
			}
			args[i] = reflect.Zero(in)
		} else {
			if inKind == reflect.Interface {
				args[i] = reflect.ValueOf(arg)
			} else if d, ok := arg.(decimal.Decimal); ok && in == decimalType {
				args[i] = reflect.ValueOf(d)
			} else if d, ok := arg.(decimal.Decimal); ok && inKind == reflect.Int {
				n, err := r.decimalToInt(node.Args[i], d)
				if err != nil {
					panic(err)
				}
				args[i] = reflect.ValueOf(n)
			} else if d, ok := arg.(int); ok && in == decimalType {
				args[i] = reflect.ValueOf(decimal.New(int64(d), 0))
			} else if html, ok := arg.(HTML); ok && inKind == reflect.String {
				args[i] = reflect.ValueOf(string(html))
			} else if reflect.TypeOf(arg).AssignableTo(in) {
				args[i] = reflect.ValueOf(arg)
			} else {
				switch inKind {
				case reflect.Int8,
					reflect.Int16,
					reflect.Int32,
					reflect.Int64,
					reflect.Uint,
					reflect.Uint8,
					reflect.Uint16,
					reflect.Uint32,
					reflect.Uint64,
					reflect.Uintptr,
					reflect.Complex64,
					reflect.Complex128,
					reflect.Ptr,
					reflect.UnsafePointer,
					reflect.Float32,
					reflect.Float64:
					return nil, fmt.Errorf("cannot use %s as function parameter type", inKind)
				}
				expectedType := typeof(reflect.Zero(in).Interface())
				return nil, r.errorf(node, "cannot use %s (type %s) as type %s in argument to %s", node.Args[i], typeof(arg), expectedType, node.Func)
			}
		}
	}

	values, err := func() (_ []reflect.Value, err error) {
		defer func() {
			if e := recover(); e != nil {
				err = r.errorf(node.Func, "%s", e)
			}
		}()
		return fun.Call(args), nil
	}()

	return values, err
}

func (r *rendering) checkBuiltInParameterCount(node *ast.Call, numIn, numOut, n int) error {
	if len(node.Args) < numIn {
		return r.errorf(node, "missing argument to %s: %s", node.Func, node)
	}
	if len(node.Args) > numIn {
		return r.errorf(node, "too many arguments to %s: %s", node.Func, node)
	}
	if numOut == 0 && n > 0 {
		return r.errorf(node, "%s used as value", node)
	}
	if n != numOut {
		return r.errorf(node, "assignment mismatch: %d variables but %d values", n, numOut)
	}
	return nil
}

// convert converts the value of expr to type typ.
func (r *rendering) convert(expr ast.Expression, typ valuetype) (interface{}, error) {
	value := asBase(r.evalExpression(expr))
	switch typ {
	case "string":
		switch v := value.(type) {
		case string:
			return value, nil
		case HTML:
			return string(v), nil
		case decimal.Decimal:
			p := v.IntPart()
			if decimal.New(p, 0).Equal(v) {
				return string(int(p)), nil
			}
		case int:
			return string(v), nil
		default:
			rv := reflect.ValueOf(v)
			if rv.Kind() == reflect.Slice {
				return convertSliceToString(rv), nil
			}
		}
	case "html":
		switch v := value.(type) {
		case string:
			return HTML(v), nil
		case HTML:
			return v, nil
		case decimal.Decimal:
			p := v.IntPart()
			if decimal.New(p, 0).Equal(v) {
				return HTML(string(int(p))), nil
			}
		case int:
			return HTML(string(v)), nil
		default:
			rv := reflect.ValueOf(v)
			if rv.Kind() == reflect.Slice {
				return HTML(convertSliceToString(rv)), nil
			}
		}
	case "number":
		switch v := value.(type) {
		case decimal.Decimal:
			return v, nil
		case int:
			return v, nil
		}
	case "int":
		switch v := value.(type) {
		case decimal.Decimal:
			return int(v.IntPart()), nil
		case int:
			return v, nil
		}
	case "map":
		if value == nil {
			return Map(nil), nil
		}
		switch v := value.(type) {
		case Map:
			return v, nil
		default:
			rv := reflect.ValueOf(v)
			switch rv.Kind() {
			case reflect.Map, reflect.Struct:
				return v, nil
			case reflect.Ptr:
				if reflect.Indirect(rv).Kind() == reflect.Struct {
					return v, nil
				}
			}
		}
	case "slice":
		if value == nil {
			return Slice(nil), nil
		}
		switch v := value.(type) {
		case string:
			return []rune(v), nil
		case HTML:
			return []rune(string(v)), nil
		case Slice:
			return v, nil
		default:
			rv := reflect.ValueOf(v)
			if rv.Kind() == reflect.Slice {
				return v, nil
			}
		}
	}
	if value == nil {
		return nil, fmt.Errorf("cannot convert nil to type %s", typ)
	}
	return nil, fmt.Errorf("cannot convert %s (type %s) to type %s", expr, typeof(value), typ)
}

// convertSliceToString converts s of type slice to a value of type string.
func convertSliceToString(s reflect.Value) string {
	l := s.Len()
	if l == 0 {
		return ""
	}
	var b strings.Builder
	b.Grow(l)
	for i := 0; i < l; i++ {
		element := asBase(s.Index(i).Interface())
		if element != nil {
			switch e := element.(type) {
			case decimal.Decimal:
				p := e.IntPart()
				if decimal.New(p, 0).Equal(e) {
					b.WriteString(string(p))
					continue
				}
			case int:
				b.WriteString(string(e))
				continue
			}
		}
		b.WriteRune(utf8.RuneError)
	}
	return b.String()
}

// isBuiltin indicates if expr is the builtin with the given name.
func (r *rendering) isBuiltin(name string, expr ast.Expression) bool {
	if n, ok := expr.(*ast.Identifier); ok {
		if n.Name != name {
			return false
		}
		for i := len(r.vars) - 1; i >= 0; i-- {
			if r.vars[i] != nil {
				if _, ok := r.vars[i][name]; ok {
					return i == 0
				}
			}
		}
	}
	return false
}

// asBase returns the base value of v.
func asBase(v interface{}) interface{} {
	switch vv := v.(type) {
	// nil
	case nil:
		return nil
	// number
	case int:
		return vv
	case uint:
		if vv > uint(maxInt) {
			return decimal.NewFromBigInt(new(big.Int).SetUint64(uint64(vv)), 0)
		}
		return int(vv)
	case int8:
		return int(vv)
	case int16:
		return int(vv)
	case int32:
		return int(vv)
	case int64:
		if vv < minInt || vv > maxInt {
			return decimal.New(vv, 0)
		}
		return int(vv)
	case uint8:
		return int(vv)
	case uint16:
		return int(vv)
	case uint32:
		if int64(vv) > maxInt {
			return decimal.New(int64(vv), 0)
		}
		return int(vv)
	case uint64:
		if vv > uint64(maxInt) {
			return decimal.NewFromBigInt(new(big.Int).SetUint64(vv), 0)
		}
		return int(vv)
	case float32:
		return decimal.NewFromFloat32(vv)
	case float64:
		return decimal.NewFromFloat(vv)
	case decimal.Decimal:
		return v
	case Numberer:
		return vv.Number()
	case complex64, complex128, uintptr:
		panic(fmt.Errorf("cannot use %T as implementation type", vv))
	// string
	case string:
		return v
	case HTML:
		return v
	case Stringer:
		return vv.String()
	// bool
	case bool:
		return v
	// Map
	case Map:
		return v
	// Slice
	case Slice:
		return v
	default:
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.String:
			return rv.String()
		case reflect.Int:
			n := rv.Int()
			if n < minInt || n > maxInt {
				return decimal.New(n, 0)
			}
			return int(n)
		case reflect.Float64:
			return rv.Float()
		case reflect.Bool:
			return rv.Bool()
		}
	}
	return v
}

// structs maintains the association between the field names of a struct,
// as they are called in the template, and the field index in the struct.
var structs = struct {
	keys map[reflect.Type]map[string]structKey
	sync.RWMutex
}{map[reflect.Type]map[string]structKey{}, sync.RWMutex{}}

// structKey represents the fields of a struct.
type structKey struct {
	isMethod bool
	index    int
}

func (sk structKey) value(st reflect.Value) interface{} {
	st = reflect.Indirect(st)
	if sk.isMethod {
		return st.Method(sk.index).Interface()
	}
	return st.Field(sk.index).Interface()
}

func structKeys(st reflect.Value) map[string]structKey {
	st = reflect.Indirect(st)
	if st.Kind() != reflect.Struct {
		return nil
	}
	structs.RLock()
	keys, ok := structs.keys[st.Type()]
	structs.RUnlock()
	if ok {
		return keys
	}
	typ := st.Type()
	structs.Lock()
	keys, ok = structs.keys[st.Type()]
	if ok {
		structs.Unlock()
		return keys
	}
	keys = map[string]structKey{}
	n := typ.NumField()
	for i := 0; i < n; i++ {
		fieldType := typ.Field(i)
		if fieldType.PkgPath != "" {
			continue
		}
		name := fieldType.Name
		if tag, ok := fieldType.Tag.Lookup("template"); ok {
			name = parseVarTag(tag)
			if name == "" {
				structs.Unlock()
				panic(fmt.Errorf("template: invalid tag of field %q", fieldType.Name))
			}
		}
		keys[name] = structKey{index: i}
	}
	n = typ.NumMethod()
	for i := 0; i < n; i++ {
		name := typ.Method(i).Name
		if _, ok := keys[name]; !ok {
			keys[name] = structKey{index: i, isMethod: true}
		}
	}
	structs.keys[st.Type()] = keys
	structs.Unlock()
	return keys
}

// parseVarTag parses the tag of a struct field and returns its name.
func parseVarTag(tag string) string {
	sp := strings.SplitN(tag, ",", 2)
	if len(sp) == 0 {
		return ""
	}
	name := sp[0]
	if name == "" {
		return ""
	}
	for _, r := range name {
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return ""
		}
	}
	return name
}
