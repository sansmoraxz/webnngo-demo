//go:build js && wasm

package main

import (
	"fmt"
	"reflect"
	"runtime"
	"syscall/js"
	"unsafe"
)

var (
	navigator = js.Global().Get("navigator")
	document  = js.Global().Get("document")

	rootElem  = document.Call("getElementById", "root")
	errorElem = document.Call("getElementById", "error")

	json = js.Global().Get("JSON")
)

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

func getML() (js.Value, error) {
	ml := navigator.Get("ml")
	if ml.Truthy() {
		return ml, nil
	}
	return js.Undefined(), fmt.Errorf("WebNN API is not available")
}

func getContext() (js.Value, error) {
	ml, err := getML()
	if err != nil {
		return js.Undefined(), err
	}

	// Create context
	contextArgs := map[string]interface{}{
		"deviceType":      "gpu",
		"powerPreference": "high-performance",
	}

	context, err := Await(ml.Call("createContext", contextArgs))
	if err != nil {
		Error(err)
		// try again with default context
		context, err = Await(ml.Call("createContext"))
		if err != nil {
			return js.Undefined(), fmt.Errorf("error creating context")
		}
	}

	return context, nil
}

func Error(err error) {
	errorElem.Call("appendChild", document.Call("createTextNode", err.Error()))
}

func main() {
	// Define the operand type
	operandType := map[string]interface{}{
		"dataType":   "float32",
		"dimensions": []any{2, 2},
	}

	context, err := getContext()
	if err != nil {
		Error(err)
		panic(err)
	}

	println("Context created")

	println("Creating graph")

	// Create a new MLGraphBuilder
	builder := js.Global().Get("MLGraphBuilder").New(context)

	// Create the constant
	constant := builder.Call("constant", map[string]interface{}{"dataType": "float32"}, sliceToTypedArray([]float32{0.2}))

	// Create inputs A and B
	A := builder.Call("input", "A", operandType)
	B := builder.Call("input", "B", operandType)

	// Create the operation C = 0.2 * A + B
	mulOp := builder.Call("mul", A, constant)
	C := builder.Call("add", mulOp, B)

	// Build the graph

	graph, err := Await(builder.Call("build", map[string]interface{}{"C": C}))
	if err != nil {
		Error(err)
		panic(err)
	}

	// these buffers will be converted to tensors by the runtime
	bufferA := sliceToTypedArray([]float32{1.0, 1.0, 1.0, 1.0})
	bufferB := sliceToTypedArray([]float32{0.8, 0.8, 0.8, 0.8})

	// for the output
	bufferC := newTypedArray(
		reflect.TypeOf(float32(0)),
		bufferA.Get("length").Int(),
	)

	s := `<div>
    <h2>Input values:</h2>
    <pre>` + jsonStringify(bufferA) + `</pre>` +
		`<pre>` + jsonStringify(bufferB) + `</pre>`

	// Create input and output maps
	inputs := map[string]interface{}{
		"A": bufferA,
		"B": bufferB,
	}
	outputs := map[string]interface{}{
		"C": bufferC,
	}

	// Compute the results
	results, err := Await(context.Call("compute", graph, inputs, outputs))
	if err != nil {
		Error(err)
		panic(err)
	}

	// Get the output value from results
	outputC := results.Get("outputs").Get("C")

	// Convert outputC to Go slice and print the values
	fmt.Println("Output value:", jsonStringify(outputC))

	js.Global().Set("outputC", outputC)

	// to dom
	s +=
		`<h2>Output value:</h2>
    <pre>` + jsonStringify(outputC) + `</pre>`

	rootElem.Set("innerHTML", s)

	// Block the main goroutine to keep the program running until the computation is complete
	select {}
}
