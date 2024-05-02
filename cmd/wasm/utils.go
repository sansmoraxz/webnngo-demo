//go:build js && wasm
// +build js,wasm

package main

import (
	"reflect"
	"runtime"
	"syscall/js"
	"unsafe"
)

func jsonStringify(v js.Value) string {
	return json.Call("stringify", v).String()
}

type numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64
}

func getJsType(v any) string {
	var jsType string
	switch any(v).(type) {
	case int8:
		jsType = "Int8Array"
	case int16:
		jsType = "Int16Array"
	case int32:
		jsType = "Int32Array"
	case int64:
		jsType = "BigInt64Array"
	case uint8:
		jsType = "Uint8Array"
	case uint16:
		jsType = "Uint16Array"
	case uint32:
		jsType = "Uint32Array"
	case uint64:
		jsType = "BigUint64Array"
	case float32:
		jsType = "Float32Array"
	case float64:
		jsType = "Float64Array"
	case int:
		jsType = "Array"
	default:
		panic("unsupported type for sliceToTypedArray")
	}
	return jsType
}

func sliceToTypedArray[T numeric](slice []T) js.Value {
	runtime.KeepAlive(slice)
	sliceLen := len(slice)

	if sliceLen == 0 {
		return js.Global().Get("Array").New()
	}

	jsType := getJsType(slice[0])
	sz := unsafe.Sizeof(slice[0])

	h := (*reflect.SliceHeader)(unsafe.Pointer(&slice))
	h.Len *= int(sz)
	h.Cap *= int(sz)
	b := *(*[]byte)(unsafe.Pointer(h))

	tmp := js.Global().Get("Uint8Array").New(len(b))
	js.CopyBytesToJS(tmp, b)

	return js.Global().Get(jsType).New(tmp.Get("buffer"), tmp.Get("byteOffset"), sliceLen)
}

func newTypedArray(t reflect.Type, length int) js.Value {
	jsType := getJsType(reflect.Zero(t).Interface())
	return js.Global().Get(jsType).New(length)
}

// Await waits for the promise to resolve and returns the result or an error.
func Await(awaitable js.Value) (js.Value, error) {
	then := make(chan []js.Value)
	defer close(then)
	thenFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		then <- args
		return nil
	})
	defer thenFunc.Release()

	catch := make(chan []js.Value)
	defer close(catch)
	catchFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		catch <- args
		return nil
	})
	defer catchFunc.Release()

	awaitable.Call("then", thenFunc).Call("catch", catchFunc)

	select {
	case result := <-then:
		return result[0], nil
	case err := <-catch:
		return js.Undefined(), js.Error{Value: err[0]}
	}
}

// Error appends an error message to the error element in the DOM.
func Error(err error) {
	errorElem.Call("appendChild", document.Call("createTextNode", err.Error()))
}
