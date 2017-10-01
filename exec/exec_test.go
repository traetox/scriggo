//
// Copyright (c) 2017 Open2b Software Snc. All Rights Reserved.
//

package exec

import (
	"bytes"
	"testing"

	"open2b/template/parser"
)

var execExprTests = []struct {
	src  string
	res  string
	vars map[string]interface{}
}{
	{`"a"`, "a", nil},
	{"`a`", "a", nil},
	{"3", "3", nil},
	{`"3"`, "3", nil},
	{"-3", "-3", nil},
	{"3.56", "3.56", nil},
	{"3.560", "3.56", nil},
	{"3.50", "3.5", nil},
	{"-3.50", "-3.5", nil},
	{"3.0", "3", nil},
	{"0.0", "0", nil},
	{"-0.0", "0", nil},
	{"true", "true", nil},
	{"false", "false", nil},
	{"_", "5", map[string]interface{}{"_": "5"}},
	{"_", "5", map[string]interface{}{"_": 5}},
	{"true", "_true_", map[string]interface{}{"true": "_true_"}},
	{"false", "_false_", map[string]interface{}{"false": "_false_"}},
	{"2 + 3", "5", nil},
	{"2 - 3", "-1", nil},
	{"2 * 3", "6", nil},
	{"2 / 3", "0.6666666666666666666666666667", nil},
	{"7 % 3", "1", nil},
	{"7.2 % 3.7", "3.5", nil},
	{"7 % 3.7", "3.3", nil},
	{"7.2 % 3", "1.2", nil},
	{"-2147483648 * -1", "2147483648", nil},                                       // math.MinInt32 * -1
	{"-2147483649 * -1", "2147483649", nil},                                       // (math.MinInt32-1) * -1
	{"2147483647 * -1", "-2147483647", nil},                                       // math.MaxInt32 * -1
	{"2147483648 * -1", "-2147483648", nil},                                       // (math.MaxInt32+1) * -1
	{"-9223372036854775808 * -1", "9223372036854775808", nil},                     // math.MinInt64 * -1
	{"-9223372036854775809 * -1", "9223372036854775809", nil},                     // (math.MinInt64-1) * -1
	{"9223372036854775807 * -1", "-9223372036854775807", nil},                     // math.MaxInt64 * -1
	{"9223372036854775808 * -1", "-9223372036854775808", nil},                     // (math.MaxInt64+1) * -1
	{"-2147483648 / -1", "2147483648", nil},                                       // math.MinInt32 / -1
	{"-2147483649 / -1", "2147483649", nil},                                       // (math.MinInt32-1) / -1
	{"2147483647 / -1", "-2147483647", nil},                                       // math.MaxInt32 / -1
	{"2147483648 / -1", "-2147483648", nil},                                       // (math.MaxInt32+1) / -1
	{"-9223372036854775808 / -1", "9223372036854775808", nil},                     // math.MinInt64 / -1
	{"-9223372036854775809 / -1", "9223372036854775809", nil},                     // (math.MinInt64-1) / -1
	{"9223372036854775807 / -1", "-9223372036854775807", nil},                     // math.MaxInt64 / -1
	{"9223372036854775808 / -1", "-9223372036854775808", nil},                     // (math.MaxInt64+1) / -1
	{"2147483647 + 2147483647", "4294967294", nil},                                // math.MaxInt32 + math.MaxInt32
	{"-2147483648 + -2147483648", "-4294967296", nil},                             // math.MinInt32 + math.MinInt32
	{"9223372036854775807 + 9223372036854775807", "18446744073709551614", nil},    // math.MaxInt64 + math.MaxInt64
	{"-9223372036854775808 + -9223372036854775808", "-18446744073709551616", nil}, // math.MinInt64 + math.MinInt64
	{"-1 + -2 * 6 / ( 6 - 1 - ( 5 * 3 ) + 2 ) * ( 1 + 2 ) * 3", "12.5", nil},
	//{"433937734937734969526500969526500", "433937734937734969526500969526500", nil},
	{"a[1]", "y", map[string]interface{}{"a": []string{"x", "y", "z"}}},
	{"a[:]", "x, y, z", map[string]interface{}{"a": []string{"x", "y", "z"}}},
	{"a[1:]", "y, z", map[string]interface{}{"a": []string{"x", "y", "z"}}},
	{"a[:2]", "x, y", map[string]interface{}{"a": []string{"x", "y", "z"}}},
	{"a[1:2]", "y", map[string]interface{}{"a": []string{"x", "y", "z"}}},
	{"a[1:3]", "y, z", map[string]interface{}{"a": []string{"x", "y", "z"}}},
	{"a[0:3]", "x, y, z", map[string]interface{}{"a": []string{"x", "y", "z"}}},
	{"a[-1:1]", "x", map[string]interface{}{"a": []string{"x", "y", "z"}}},
	{"a[1:10]", "y, z", map[string]interface{}{"a": []string{"x", "y", "z"}}},
	{"a[2:2]", "", map[string]interface{}{"a": []string{"x", "y", "z"}}},
	{"a[2:1]", "", map[string]interface{}{"a": []string{"x", "y", "z"}}},
	{"a[:]", "x€z", map[string]interface{}{"a": "x€z"}},
	{"a[1:]", "€z", map[string]interface{}{"a": "x€z"}},
	{"a[:2]", "x€", map[string]interface{}{"a": "x€z"}},
	{"a[1:2]", "€", map[string]interface{}{"a": "x€z"}},
	{"a[1:3]", "€z", map[string]interface{}{"a": "x€z"}},
	{"a[0:3]", "x€z", map[string]interface{}{"a": "x€z"}},
	{"a[1:]", "xz", map[string]interface{}{"a": "€xz"}},
	{"a[:2]", "xz", map[string]interface{}{"a": "xz€"}},
	{"a[-1:1]", "x", map[string]interface{}{"a": "xz€"}},
	{"a[1:10]", "z€", map[string]interface{}{"a": "xz€"}},
	{"a[2:2]", "", map[string]interface{}{"a": "xz€"}},
	{"a[2:1]", "", map[string]interface{}{"a": "xz€"}},

	// ==, !=
	{"true == true", "true", nil},
	{"false == false", "true", nil},
	{"true == false", "false", nil},
	{"false == true", "false", nil},
	{"true != true", "false", nil},
	{"false != false", "false", nil},
	{"true != false", "true", nil},
	{"false != true", "true", nil},
	{"a == nil", "true", map[string]interface{}{"a": map[string]interface{}(nil)}},
	{"a != nil", "false", map[string]interface{}{"a": map[string]interface{}(nil)}},
	{"nil == a", "true", map[string]interface{}{"a": map[string]interface{}(nil)}},
	{"nil != a", "false", map[string]interface{}{"a": map[string]interface{}(nil)}},

	// len
	{"len()", "0", nil},
	{"len(``)", "0", nil},
	{"len(`a`)", "1", nil},
	{"len(`abc`)", "3", nil},
	{"len(`€`)", "1", nil},
	{"len(`€`)", "1", nil},

	// join
	{"join()", "", nil},
	{"join(a, ``)", "", map[string]interface{}{"a": []string(nil)}},
	{"join(a, ``)", "", map[string]interface{}{"a": []string{}}},
	{"join(a, ``)", "a", map[string]interface{}{"a": []string{"a"}}},
	{"join(a, ``)", "ab", map[string]interface{}{"a": []string{"a", "b"}}},
	{"join(a, `,`)", "a,b,c", map[string]interface{}{"a": []string{"a", "b", "c"}}},

	// contains
	{"contains()", "true", nil},
	{"contains(``,``)", "true", nil},
	{"contains(`a`,``)", "true", nil},
	{"contains(`abc`,`b`)", "true", nil},
	{"contains(`abc`,`e`)", "false", nil},

	// hasPrefix
	{"hasPrefix()", "true", nil},
	{"hasPrefix(``,``)", "true", nil},
	{"hasPrefix(`a`,``)", "true", nil},
	{"hasPrefix(`abc`,`a`)", "true", nil},
	{"hasPrefix(`abc`,`b`)", "false", nil},

	// hasSuffix
	{"hasSuffix()", "true", nil},
	{"hasSuffix(``,``)", "true", nil},
	{"hasSuffix(`a`,``)", "true", nil},
	{"hasSuffix(`abc`,`c`)", "true", nil},
	{"hasSuffix(`abc`,`b`)", "false", nil},

	// index
	{"index()", "0", nil},
	{"index(``,``)", "0", nil},
	{"index(`a`,``)", "0", nil},
	{"index(`ab€c`,`a`)", "0", nil},
	{"index(`ab€c`,`b`)", "1", nil},
	{"index(`ab€c`,`€`)", "2", nil},
	{"index(`ab€c`,`c`)", "3", nil},

	// repeat
	{"repeat()", "", nil},
	{"repeat(``)", "", nil},
	{"repeat(``, 0)", "", nil},
	{"repeat(`a`, 0)", "", nil},
	{"repeat(`a`, 1)", "a", nil},
	{"repeat(`a`, 5)", "aaaaa", nil},
	{"repeat(`€`, 3)", "€€€", nil},

	// replace
	{"replace()", "", nil},
	{"replace(``)", "", nil},
	{"replace(``, ``)", "", nil},
	{"replace(``, ``, ``)", "", nil},
	{"replace(`abc`, `b`, `e`)", "aec", nil},
	{"replace(`abc`, `b`, `€`)", "a€c", nil},
	{"replace(`abcbcba`, `b`, `e`)", "aececea", nil},

	// split
	{"split()", "", nil},
	{"split(``, ``)", "", nil},
	{"split(`a`, ``)", "a", nil},
	{"split(`ab`, ``)", "a, b", nil},
	{"split(`a,b,c`, `,`)", "a, b, c", nil},
	{"split(`a,b,c,`, `,`)", "a, b, c, ", nil},

	// toLower
	{"toLower()", "", nil},
	{"toLower(``)", "", nil},
	{"toLower(`a`)", "a", nil},
	{"toLower(`A`)", "a", nil},
	{"toLower(`aB`)", "ab", nil},
	{"toLower(`aBCd`)", "abcd", nil},
	{"toLower(`èÈ`)", "èè", nil},

	// toUpper
	{"toUpper()", "", nil},
	{"toUpper(``)", "", nil},
	{"toUpper(`A`)", "A", nil},
	{"toUpper(`a`)", "A", nil},
	{"toUpper(`Ab`)", "AB", nil},
	{"toUpper(`AbcD`)", "ABCD", nil},
	{"toUpper(`Èè`)", "ÈÈ", nil},

	// trimSpace
	{"trimSpace()", "", nil},
	{"trimSpace(``)", "", nil},
	{"trimSpace(` `)", "", nil},
	{"trimSpace(` a`)", "a", nil},
	{"trimSpace(`a `)", "a", nil},
	{"trimSpace(` a `)", "a", nil},
	{"trimSpace(` a b  `)", "a b", nil},

	// min
	{"min()", "0", nil},
	{"min(0)", "0", nil},
	{"min(0, 0)", "0", nil},
	{"min(5, 7)", "5", nil},
	{"min(7, 5)", "5", nil},
	{"min(-7, 5)", "-7", nil},
	{"min(7, -5)", "-5", nil},

	// max
	{"max()", "0", nil},
	{"max(0)", "0", nil},
	{"max(0, 0)", "0", nil},
	{"max(5, 7)", "7", nil},
	{"max(7, 5)", "7", nil},
	{"max(-7, 5)", "5", nil},
	{"max(7, -5)", "7", nil},

	// abs
	{"abs()", "0", nil},
	{"abs(0)", "0", nil},
	{"abs(1)", "1", nil},
	{"abs(-1)", "1", nil},
	{"abs(3.56)", "3.56", nil},
	{"abs(-3.56)", "3.56", nil},

	// round
	{"round()", "0", nil},
	{"round(0)", "0", nil},
	{"round(5.3752, 2)", "5.38", nil},

	// int
	{"int()", "0", nil},
	{"int(0)", "0", nil},
	{"int(1)", "1", nil},
	{"int(0.5)", "0", nil},
	{"int(-0.5)", "0", nil},
	{"int(3.56)", "3", nil},
}

var execStmtTests = []struct {
	src  string
	res  string
	vars map[string]interface{}
}{
	{"{% for p in products %}{{ p }}\n{% end %}", "a\nb\nc\n",
		map[string]interface{}{"products": []string{"a", "b", "c"}}},
	{"{% for i, p in products %}{{ i }}: {{ p }}\n{% end %}", "0: a\n1: b\n2: c\n",
		map[string]interface{}{"products": []string{"a", "b", "c"}}},
}

func TestExecExpressions(t *testing.T) {
	for _, expr := range execExprTests {
		var tree, err = parser.Parse([]byte("{{" + expr.src + "}}"))
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		var env = NewEnv(tree, nil)
		err = env.Execute(b, expr.vars)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var res = b.String()
		if res != expr.res {
			t.Errorf("source: %q, unexpected %q, expecting %q\n", expr.src, res, expr.res)
		}
	}
}

func TestExecStatements(t *testing.T) {
	for _, stmt := range execStmtTests {
		var tree, err = parser.Parse([]byte(stmt.src))
		if err != nil {
			t.Errorf("source: %q, %s\n", stmt.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		var env = NewEnv(tree, nil)
		err = env.Execute(b, stmt.vars)
		if err != nil {
			t.Errorf("source: %q, %s\n", stmt.src, err)
			continue
		}
		var res = b.String()
		if res != stmt.res {
			t.Errorf("source: %q, unexpected %q, expecting %q\n", stmt.src, res, stmt.res)
		}
	}
}
