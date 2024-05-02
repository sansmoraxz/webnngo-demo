package main

import (
	"syscall/js"
)

func jsonStringify(v js.Value) string {
    return json.Call("stringify", v).String()
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
