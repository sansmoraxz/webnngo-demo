package main

// go:generate gopherjs build main.go -o main.js

import (
	"fmt"
	"reflect"
	"runtime"
	"syscall/js"
	"unsafe"
)

func jsonStringify(v js.Value) string {
    return js.Global().Get("JSON").Call("stringify", v).String()
}

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


func main() {
    // Define the operand type
    operandType := map[string]interface{}{
        "dataType":   "float32",
        "dimensions": []any{2, 2},
    }

    // panic("This is a panic")

    // Get the navigator.ml object
    navigator := js.Global().Get("navigator")
    // display the navigator object
    fmt.Printf("Navigator: %#v\n", jsonStringify(navigator))

    ml := navigator.Get("ml")

    // Create context
    contextArgs := map[string]interface{}{
        "deviceType": "gpu",
        "powerPreference": "high-performance",
    }

    createContextPromise := ml.Call("createContext", contextArgs)
    contextChan := make(chan js.Value)
    createContextPromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
        contextChan <- args[0]
        return nil
    }))

    // Get the context value
    context := <-contextChan

    println("Context created")
    println("Context: ", jsonStringify(context))
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
    buildPromise := builder.Call("build", map[string]interface{}{"C": C})
    graphChan := make(chan js.Value)
    buildPromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
        graphChan <- args[0]
        return nil
    }))

    // Get the graph value
    graph := <-graphChan

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


    doc := js.Global().Get("document")

    div := doc.Call("getElementById", "root")
    div.Set("innerHTML", s)

    // Block the main goroutine to keep the program running until the computation is complete
    select {}
}
