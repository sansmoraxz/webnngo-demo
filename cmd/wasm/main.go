package main

// go:generate gopherjs build main.go -o main.js

import (
	"fmt"
	"reflect"
	"runtime"
	"syscall/js"
	"unsafe"
)

var (
    navigator = js.Global().Get("navigator")
    document = js.Global().Get("document")

    json = js.Global().Get("JSON")
)

type numeric interface {
    ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64
}

func sliceToTypedArray[T numeric](slice []T) js.Value {
    runtime.KeepAlive(slice)
    sliceLen := len(slice)
    var jsType string
    sz := unsafe.Sizeof(slice[0])
    switch any(slice[0]).(type) {
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

    h := (*reflect.SliceHeader)(unsafe.Pointer(&slice))
    h.Len *= int(sz)
    h.Cap *= int(sz)
    b := *(*[]byte)(unsafe.Pointer(h))

    tmp := js.Global().Get("Uint8Array").New(len(b))
    js.CopyBytesToJS(tmp, b)

    return js.Global().Get(jsType).New(tmp.Get("buffer"), tmp.Get("byteOffset"), sliceLen)
}

func getContext() (js.Value, error) {
    ml := navigator.Get("ml")

    // Create context
    contextArgs := map[string]interface{}{
        "deviceType": "gpu",
        "powerPreference": "high-performance",
    }

    context, err := Await(ml.Call("createContext", contextArgs))
    if err != nil {
        println("GPU context creation failed: ", err)
        // try again with default context
        context, err = Await(ml.Call("createContext"))
        if err != nil {
            return js.Undefined(), fmt.Errorf("error creating context")
        }
    }

    return context, nil
}


func main() {
    // Define the operand type
    operandType := map[string]interface{}{
        "dataType":   "float32",
        "dimensions": []any{2, 2},
    }

    context, err := getContext()
    if err != nil {
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
        panic(err)
    }

    // Prepare inputs A and B
    bufferA := sliceToTypedArray([]float32{1.0, 1.0, 1.0, 1.0})
    bufferB := sliceToTypedArray([]float32{0.8, 0.8, 0.8, 0.8})
    s := `<div>
    <h1>Input values:</h1>
    <pre>` + jsonStringify(bufferA) + `</pre>` +
    `<pre>` + jsonStringify(bufferB) + `</pre>`
    bufferC := sliceToTypedArray([]float32{0, 0, 0, 0})

    // Create input and output maps
    inputs := map[string]interface{}{
        "A": bufferA,
        "B": bufferB,
    }
    outputs := map[string]interface{}{
        "C": bufferC,
    }

    // Compute the results
    computePromise := context.Call("compute", graph, inputs, outputs)
    computeChan := make(chan js.Value)
    computePromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
        computeChan <- args[0]
        return nil
    }))

    // Get the results
    results := <-computeChan

    // Get the output value from results
    outputC := results.Get("outputs").Get("C")

    // Convert outputC to Go slice and print the values
    fmt.Println("Output value:", jsonStringify(outputC))

    js.Global().Set("outputC", outputC)

    // to dom
    s +=
    `<h1>Output value:</h1>
    <pre>` + jsonStringify(outputC) + `</pre>`


    div := document.Call("getElementById", "root")
    div.Set("innerHTML", s)

    // Block the main goroutine to keep the program running until the computation is complete
    select {}
}
