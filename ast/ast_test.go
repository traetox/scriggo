//
// Copyright (c) 2017 Open2b Software Snc. All Rights Reserved.
//

package ast

import (
	"testing"

	"github.com/shopspring/decimal"
)

var n1 = NewDecimal(nil, decimal.New(1, 0))
var n2 = NewDecimal(nil, decimal.New(2, 0))
var n3 = NewDecimal(nil, decimal.New(3, 0))
var n5 = NewDecimal(nil, decimal.New(5, 0))

var expressionStringTests = []struct {
	str  string
	expr Expression
}{
	{"1", n1},
	{"3.59", NewDecimal(nil, decimal.NewFromFloat(3.59))},
	{`"abc"`, NewString(nil, "abc")},
	{"\"a\\tb\"", NewString(nil, "a\tb")},
	{"x", NewIdentifier(nil, "x")},
	{"-1", NewUnaryOperator(nil, OperatorSubtraction, n1)},
	{"1+2", NewBinaryOperator(nil, OperatorAddition, n1, n2)},
	{"1+2", NewBinaryOperator(nil, OperatorAddition, n1, n2)},
	{"f()", NewCall(nil, NewIdentifier(nil, "f"), []Expression{})},
	{"f(a)", NewCall(nil, NewIdentifier(nil, "f"), []Expression{NewIdentifier(nil, "a")})},
	{"f(a,b)", NewCall(nil, NewIdentifier(nil, "f"), []Expression{NewIdentifier(nil, "a"), NewIdentifier(nil, "b")})},
	{"a[2]", NewIndex(nil, NewIdentifier(nil, "a"), n2)},
	{"a[:]", NewSlice(nil, NewIdentifier(nil, "a"), nil, nil)},
	{"a[2:]", NewSlice(nil, NewIdentifier(nil, "a"), n2, nil)},
	{"a[:5]", NewSlice(nil, NewIdentifier(nil, "a"), nil, n5)},
	{"a[2:5]", NewSlice(nil, NewIdentifier(nil, "a"), n2, n5)},
	{"a.b", NewSelector(nil, NewIdentifier(nil, "a"), "b")},
	{"(a)", NewParentesis(nil, NewIdentifier(nil, "a"))},
	{"-(1+2)", NewUnaryOperator(nil, OperatorSubtraction, NewBinaryOperator(nil, OperatorAddition, n1, n2))},
	{"-(+1)", NewUnaryOperator(nil, OperatorSubtraction, NewUnaryOperator(nil, OperatorAddition, n1))},
	{"1*2+-3", NewBinaryOperator(nil, OperatorAddition,
		NewBinaryOperator(nil, OperatorMultiplication, n1, n2),
		NewUnaryOperator(nil, OperatorSubtraction, n3))},
	{"f()-2", NewBinaryOperator(nil, OperatorSubtraction, NewCall(nil, NewIdentifier(nil, "f"), []Expression{}), n2)},
	{"-a.b", NewUnaryOperator(nil, OperatorSubtraction, NewSelector(nil, NewIdentifier(nil, "a"), "b"))},
}

func TestExpressionString(t *testing.T) {
	for _, e := range expressionStringTests {
		if e.expr.String() != e.str {
			t.Errorf("unexpected %q, expecting %q\n", e.expr.String(), e.str)
		}
	}
}
