# Process uppercase example

Build this one-shot guest as a Windows PE executable and package it with a Node Contract using ABI `{ "kind": "process", "version": "v1" }`. Yotta launches it only with `--yotta-plugin-process-v1` inside the process sandbox.

The example receives a typed string Value Envelope, uppercases the inline JSON carrier, preserves the exact resolved type, and emits the declared `result` port.

