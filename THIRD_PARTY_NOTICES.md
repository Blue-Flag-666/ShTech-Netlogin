# Third-party notices

The embedded captcha model and character set are derived from
[`ShanghaitechGeekPie/net-loginer`](https://github.com/ShanghaitechGeekPie/net-loginer),
licensed under the MIT License. Its license text is included in
`THIRD_PARTY_NET_LOGINER_LICENSE`.

The program uses
[`Kazuhito00/onnx-purego-interpreter`](https://github.com/Kazuhito00/onnx-purego-interpreter),
licensed under the MIT License. Its license text is included in
`THIRD_PARTY_ONNX_INTERPRETER_LICENSE`. The vendored dependency contains a
small local patch that adds bidirectional LSTM inference, required by the
captcha model. Re-running `go mod vendor` will overwrite that patch.
