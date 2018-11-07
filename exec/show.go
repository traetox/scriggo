//
// Copyright (c) 2017-2018 Open2b Software Snc. All Rights Reserved.
//

package exec

import (
	"bytes"
	"fmt"
	"html"
	"reflect"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
)

func interfaceToText(expr interface{}, version string) string {

	if expr == nil {
		return ""
	}

	var s string

	switch e := expr.(type) {
	case string:
		s = e
	case HTML:
		s = string(e)
	case int:
		s = strconv.Itoa(e)
	case decimal.Decimal:
		s = e.String()
	case bool:
		if e {
			s = "true"
		} else {
			s = "false"
		}
	case []string:
		st := make([]string, len(e))
		for i, v := range e {
			st[i] = v
		}
		s = strings.Join(st, ", ")
	case []HTML:
		st := make([]string, len(e))
		for i, h := range e {
			st[i] = string(h)
		}
		s = strings.Join(st, ", ")
	case []int:
		buf := make([]string, len(e))
		for i, n := range e {
			buf[i] = strconv.Itoa(n)
		}
		s = strings.Join(buf, ", ")
	case []bool:
		buf := make([]string, len(e))
		for i, b := range e {
			if b {
				buf[i] = "true"
			} else {
				buf[i] = "false"
			}
		}
		s = strings.Join(buf, ", ")
	default:
		if str, ok := e.(fmt.Stringer); ok {
			s = str.String()
		}
	}

	return s
}

func interfaceToHTML(expr interface{}, version string) string {

	if expr == nil {
		return ""
	}

	var s string

	switch e := expr.(type) {
	case string:
		s = html.EscapeString(e)
	case HTML:
		s = string(e)
	case int:
		s = strconv.Itoa(e)
	case decimal.Decimal:
		s = e.String()
	case bool:
		if e {
			s = "true"
		} else {
			s = "false"
		}
	case []string:
		st := make([]string, len(e))
		for i, v := range e {
			st[i] = html.EscapeString(v)
		}
		s = strings.Join(st, ", ")
	case []HTML:
		st := make([]string, len(e))
		for i, h := range e {
			st[i] = string(h)
		}
		s = strings.Join(st, ", ")
	case []int:
		buf := make([]string, len(e))
		for i, n := range e {
			buf[i] = strconv.Itoa(n)
		}
		s = strings.Join(buf, ", ")
	case []bool:
		buf := make([]string, len(e))
		for i, b := range e {
			if b {
				buf[i] = "true"
			} else {
				buf[i] = "false"
			}
		}
		s = strings.Join(buf, ", ")
	default:
		if str, ok := e.(fmt.Stringer); ok {
			s = html.EscapeString(str.String())
		}
	}

	return s
}

func interfaceToCSS(expr interface{}, version string) string {
	return interfaceToText(expr, version)
}

var mapStringToInterfaceType = reflect.TypeOf(map[string]interface{}{})

func interfaceToJavaScript(expr interface{}, version string) string {

	if expr == nil {
		return "null"
	}

	switch e := expr.(type) {
	case string:
		return stringToJavaScript(e)
	case HTML:
		return stringToJavaScript(string(e))
	case int:
		return strconv.Itoa(e)
	case decimal.Decimal:
		return e.String()
	case bool:
		if e {
			return "true"
		}
		return "false"
	case map[string]interface{}:
		return mapToJavaScript(e, version)
	case []string:
		if e == nil {
			return "null"
		}
		var s string
		for i, t := range e {
			if i > 0 {
				s += ","
			}
			s += stringToJavaScript(t)
		}
		return "[" + s + "]"
	case []HTML:
		if e == nil {
			return "null"
		}
		var s string
		for i, t := range e {
			if i > 0 {
				s += ","
			}
			s += stringToJavaScript(string(t))
		}
		return "[" + s + "]"
	case []int:
		if e == nil {
			return "null"
		}
		var s string
		for i, n := range e {
			if i > 0 {
				s += ","
			}
			s += strconv.Itoa(n)
		}
		return "[" + s + "]"
	case []bool:
		if e == nil {
			return "null"
		}
		buf := make([]string, len(e))
		for i, b := range e {
			if b {
				buf[i] = "true"
			} else {
				buf[i] = "false"
			}
		}
		return "[" + strings.Join(buf, ",") + "]"
	case []map[string]interface{}:
		if e == nil {
			return "null"
		}
		buf := make([]string, len(e))
		for i, v := range e {
			buf[i] = mapToJavaScript(v, version)
		}
		return "[" + strings.Join(buf, ",") + "]"
	default:
		v := reflect.ValueOf(e)
		if !v.IsValid() {
			return ""
		}
		switch v.Kind() {
		case reflect.Slice:
			if v.IsNil() {
				return "null"
			}
			if l := v.Len(); l > 0 {
				s := "["
				for i := 0; i < l; i++ {
					if i > 0 {
						s += ","
					}
					s += interfaceToJavaScript(v.Index(i).Interface(), version)
				}
				return s + "]"
			}
			return "[]"
		case reflect.Struct:
			return structToJavaScript(v.Type(), v, version)
		case reflect.Map:
			if !v.Type().ConvertibleTo(mapStringToInterfaceType) {
				return "null"
			}
			return interfaceToJavaScript(v.Convert(mapStringToInterfaceType).Interface(), version)
		case reflect.Ptr:
			t := v.Type().Elem()
			if t.Kind() != reflect.Struct {
				return ""
			}
			v = v.Elem()
			if !v.IsValid() {
				return "null"
			}
			return structToJavaScript(t, v, version)
		default:
			return "undefined"
		}
	}

	return ""
}

const hexchars = "0123456789abcdef"

func stringToJavaScript(s string) string {
	if len(s) == 0 {
		return "\"\""
	}
	var b bytes.Buffer
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString("\\\\")
		case '"':
			b.WriteString("\\\"")
		case '\n':
			b.WriteString("\\n")
		case '\r':
			b.WriteString("\\r")
		case '\t':
			b.WriteString("\\t")
		case '\u2028':
			b.WriteString("\\u2028")
		case '\u2029':
			b.WriteString("\\u2029")
		default:
			if 0 <= r && r <= 31 || r == '<' || r == '>' || r == '&' {
				b.WriteString("\\x")
				b.WriteByte(hexchars[r>>4])
				b.WriteByte(hexchars[r&0xF])
			} else {
				b.WriteRune(r)
			}
		}
	}
	return "\"" + b.String() + "\""
}

func structToJavaScript(t reflect.Type, v reflect.Value, version string) string {
	var s string
	for _, field := range getStructFields(v) {
		if field.version == "" || field.version == version {
			if len(s) > 0 {
				s += ","
			}
			s += stringToJavaScript(field.name) + ":" + interfaceToJavaScript(v.Field(field.index).Interface(), version)
		}
	}
	return "{" + s + "}"
}

func mapToJavaScript(e map[string]interface{}, version string) string {
	if e == nil {
		return "null"
	}
	var s string
	for k, v := range e {
		if len(s) > 0 {
			s += ","
		}
		s += stringToJavaScript(k) + ":" + interfaceToJavaScript(v, version)
	}
	return "{" + s + "}"
}
