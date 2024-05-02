//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"reflect"
	"syscall/js"
)

var (
	navigator = js.Global().Get("navigator")
	document  = js.Global().Get("document")

	rootElem  = document.Call("getElementById", "root")
	errorElem = document.Call("getElementById", "error")

	json = js.Global().Get("JSON")
)

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

func buildGraph(context js.Value, operandType map[string]interface{}) (js.Value, error) {
	builder := js.Global().Get("MLGraphBuilder").New(context)

	constant := builder.Call("constant", map[string]interface{}{"dataType": "float32"}, sliceToTypedArray([]float32{0.2}))
	
	// Create the operation C = 0.2 * A + B

	A := builder.Call("input", "A", operandType)
	B := builder.Call("input", "B", operandType)
	mulOp := builder.Call("mul", A, constant)

	C := builder.Call("add", mulOp, B)

	graph, err := Await(builder.Call("build", map[string]interface{}{"C": C}))
	if err != nil {
		Error(err)
		return js.Undefined(), fmt.Errorf("error building graph")
	}
	return graph, nil
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
	graph, err := buildGraph(context, operandType)
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
