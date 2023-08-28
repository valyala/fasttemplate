// +build !appengine

package fasttemplate

import (
	"reflect"
	"unsafe"
)

func unsafeBytes2String(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func unsafeString2Bytes(s string) (b []byte) {
	return *(*[]byte)(unsafe.Pointer(&s))
}
