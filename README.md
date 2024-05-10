# WebNN API Demo for Golang

Neural networks for browsers through experimental WebNN API.

This is a simple golang implementation of the WebNN API. It will try to build the graph through the GPU api. In case of error, it will perform the calculations in CPU.

### Requirements

Please ensure that you have WebNN api enabled in your browser.

For chromium browsers you may do it through `chrome://flags/#web-machine-learning-neural-network` and change the value to `Enabled`. It is not yet available in firefox though.

GPU is running in Windows 11 or Linux (with WebGPU enabled). Otherwise it will fallback to using CPU.

### Conclusion:

WebNN API is still WIP and not yet available to all browsers. Check and enable necessary flags in your browser to test the WebNN API (if available).

Also this repo is mostly using js glue code to interact with the WebNN API. You may be better off to use plain JS at least until native WASM api is available.


### References:

- [WebNN API](https://webmachinelearning.github.io/webnn/)
- [WebNN API Spec](https://webmachinelearning.github.io/webnn/#webnn-api)
- [Chromium Implementation](https://docs.google.com/document/d/1KDVuz38fx3SpLVdE8FzCCqASjFfOBXcJWj124jP7ZZ4/edit#heading=h.7nki9mck5t64)
