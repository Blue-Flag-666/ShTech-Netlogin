# ONNX interpreter patch

`vendor/github.com/Kazuhito00/onnx-purego-interpreter/internal/ops/lstm.go`
has been extended to support ONNX `direction="bidirectional"` and
`direction="reverse"` LSTM tensors.

The upstream revision used by this project only handles the forward direction.
The embedded captcha model contains a bidirectional LSTM, so the unmodified
interpreter constructs tensors with mismatched shapes at runtime.

This patch was checked using the captcha image and successful `validCode` pair
from the supplied HAR. No credentials, cookies, tokens, session IDs, or HAR
content are stored in this repository.
