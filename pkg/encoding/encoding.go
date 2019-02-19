// Go version: go1.11.5

package encoding

import original "encoding"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"BinaryMarshaler": reflect.TypeOf((*original.BinaryMarshaler)(nil)).Elem(),
	"BinaryUnmarshaler": reflect.TypeOf((*original.BinaryUnmarshaler)(nil)).Elem(),
	"TextMarshaler": reflect.TypeOf((*original.TextMarshaler)(nil)).Elem(),
	"TextUnmarshaler": reflect.TypeOf((*original.TextUnmarshaler)(nil)).Elem(),
}
