package captcha

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"

	"github.com/Kazuhito00/onnx-purego-interpreter/onnx"
	"github.com/Kazuhito00/onnx-purego-interpreter/tensor"
	"golang.org/x/image/draw"
)

// The model and charset originate from ShanghaitechGeekPie/net-loginer (MIT).
// See THIRD_PARTY_NET_LOGINER_LICENSE.
//
//go:embed assets/shtu_captcha.onnx
var modelBytes []byte

//go:embed assets/charset.json
var charsetBytes []byte

type Recognizer struct {
	session *onnx.Session
	charset []string
}

func New() (*Recognizer, error) {
	var charset []string
	if err := json.Unmarshal(charsetBytes, &charset); err != nil {
		return nil, fmt.Errorf("加载验证码字符表: %w", err)
	}
	session, err := onnx.NewSession(modelBytes)
	if err != nil {
		return nil, fmt.Errorf("加载验证码模型: %w", err)
	}
	return &Recognizer{session: session, charset: charset}, nil
}

func (r *Recognizer) Recognize(encoded []byte) (string, error) {
	source, _, err := image.Decode(bytes.NewReader(encoded))
	if err != nil {
		return "", fmt.Errorf("解码图片: %w", err)
	}
	bounds := source.Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return "", fmt.Errorf("验证码图片尺寸无效")
	}

	height := 64
	width := int(float64(bounds.Dx())*float64(height)/float64(bounds.Dy()) + 0.5)
	if width < 1 {
		width = 1
	}
	resized := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(resized, resized.Bounds(), source, bounds, draw.Over, nil)

	data := make([]float32, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r16, g16, b16, _ := resized.At(x, y).RGBA()
			gray := 0.2989*float32(r16>>8) + 0.5870*float32(g16>>8) + 0.1140*float32(b16>>8)
			data[y*width+x] = (gray/255 - 0.456) / 0.224
		}
	}

	outputs, err := r.session.Run(tensor.NewDense[float32](tensor.Shape{1, 1, height, width}, data))
	if err != nil {
		return "", fmt.Errorf("运行验证码模型: %w", err)
	}
	for _, output := range outputs {
		values, ok := output.(*tensor.Dense[int64])
		if !ok {
			continue
		}
		return r.decodeCTC(values.Data())
	}
	return "", fmt.Errorf("验证码模型没有返回 int64 输出")
}

func (r *Recognizer) decodeCTC(values []int64) (string, error) {
	result := ""
	var last int64
	for _, value := range values {
		if value == 0 || value == last {
			continue
		}
		if value < 0 || int(value) >= len(r.charset) {
			return "", fmt.Errorf("验证码模型返回越界字符索引 %d", value)
		}
		last = value
		result += r.charset[value]
	}
	return result, nil
}
