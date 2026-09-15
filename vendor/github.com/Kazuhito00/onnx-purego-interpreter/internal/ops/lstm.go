package ops

import (
	"math"

	"github.com/Kazuhito00/onnx-purego-interpreter/internal/ir"
	"github.com/Kazuhito00/onnx-purego-interpreter/tensor"
)

// LSTM op — Long Short-Term Memory (recurrent).
// Inputs: X[seq_len, batch, input_size], W[num_dir, 4*hidden, input], R[num_dir, 4*hidden, hidden],
//
//	B[num_dir, 8*hidden] (optional), sequence_lens (optional),
//	initial_h[num_dir, batch, hidden] (optional), initial_c[num_dir, batch, hidden] (optional)
//
// Outputs: Y[seq_len, num_dir, batch, hidden], Y_h[num_dir, batch, hidden], Y_c[num_dir, batch, hidden]
func opLSTM(node *ir.Node, inputs []tensor.Tensor) ([]tensor.Tensor, error) {
	direction := node.GetAttrString("direction", "forward")
	hiddenSize := int(node.GetAttrInt("hidden_size", 0))
	x := inputs[0].(*tensor.Dense[float32])
	w := inputs[1].(*tensor.Dense[float32])
	r := inputs[2].(*tensor.Dense[float32])
	xShape := x.Shape()
	seqLen := xShape[0]
	batch := xShape[1]
	inputSize := xShape[2]
	if hiddenSize == 0 {
		hiddenSize = r.Shape()[2]
	}
	H := hiddenSize
	gate4H := 4 * H
	numDir := w.Shape()[0]

	var biasData []float32
	if len(inputs) > 3 && inputs[3] != nil {
		biasData = inputs[3].(*tensor.Dense[float32]).Data()
	}
	var hData []float32
	if len(inputs) > 5 && inputs[5] != nil {
		h := inputs[5].(*tensor.Dense[float32])
		hData = make([]float32, h.Len())
		copy(hData, h.Data())
	} else {
		hData = make([]float32, numDir*batch*H)
	}
	var cData []float32
	if len(inputs) > 6 && inputs[6] != nil {
		c := inputs[6].(*tensor.Dense[float32])
		cData = make([]float32, c.Len())
		copy(cData, c.Data())
	} else {
		cData = make([]float32, numDir*batch*H)
	}
	wData := w.Data()
	rData := r.Data()
	xData := x.Data()
	wDirStride := gate4H * inputSize
	rDirStride := gate4H * H
	bDirStride := 2 * gate4H
	yData := make([]float32, seqLen*numDir*batch*H)

	for dir := 0; dir < numDir; dir++ {
		wBase := dir * wDirStride
		rBase := dir * rDirStride
		bBase := dir * bDirStride
		stateBase := dir * batch * H
		backward := direction == "reverse" || (direction == "bidirectional" && dir == 1)
		for step := 0; step < seqLen; step++ {
			t := step
			if backward {
				t = seqLen - 1 - step
			}
			for b := 0; b < batch; b++ {
				xOff := t*batch*inputSize + b*inputSize
				hOff := stateBase + b*H
				gates := make([]float32, gate4H)
				for g := 0; g < gate4H; g++ {
					var sum float32
					wOff := wBase + g*inputSize
					for j := 0; j < inputSize; j++ {
						sum += wData[wOff+j] * xData[xOff+j]
					}
					gates[g] = sum
				}
				for g := 0; g < gate4H; g++ {
					var sum float32
					rOff := rBase + g*H
					for j := 0; j < H; j++ {
						sum += rData[rOff+j] * hData[hOff+j]
					}
					gates[g] += sum
				}
				if biasData != nil {
					for g := 0; g < gate4H; g++ {
						gates[g] += biasData[bBase+g] + biasData[bBase+gate4H+g]
					}
				}
				for h := 0; h < H; h++ {
					it := sigmoid32(gates[h])
					ot := sigmoid32(gates[H+h])
					ft := sigmoid32(gates[2*H+h])
					ct := float32(math.Tanh(float64(gates[3*H+h])))
					cellIdx := hOff + h
					cData[cellIdx] = ft*cData[cellIdx] + it*ct
					hData[cellIdx] = ot * float32(math.Tanh(float64(cData[cellIdx])))
				}
				yOff := t*numDir*batch*H + dir*batch*H + b*H
				copy(yData[yOff:yOff+H], hData[hOff:hOff+H])
			}
		}
	}
	Y := tensor.NewDense[float32](tensor.Shape{seqLen, numDir, batch, H}, yData)
	Yh := tensor.NewDense[float32](tensor.Shape{numDir, batch, H}, hData)
	Yc := tensor.NewDense[float32](tensor.Shape{numDir, batch, H}, cData)
	return []tensor.Tensor{Y, Yh, Yc}, nil
}

func sigmoid32(x float32) float32 {
	return float32(1.0 / (1.0 + math.Exp(-float64(x))))
}

// GRU op — Gated Recurrent Unit.
// Inputs: X[seq_len, batch, input_size], W[num_dir, 3*hidden, input], R[num_dir, 3*hidden, hidden],
//
//	B[num_dir, 6*hidden] (optional), sequence_lens (optional),
//	initial_h[num_dir, batch, hidden] (optional)
//
// Outputs: Y[seq_len, num_dir, batch, hidden], Y_h[num_dir, batch, hidden]
func opGRU(node *ir.Node, inputs []tensor.Tensor) ([]tensor.Tensor, error) {
	hiddenSize := int(node.GetAttrInt("hidden_size", 0))
	linearBeforeReset := node.GetAttrInt("linear_before_reset", 0) != 0

	x := inputs[0].(*tensor.Dense[float32])
	w := inputs[1].(*tensor.Dense[float32])
	r := inputs[2].(*tensor.Dense[float32])

	xShape := x.Shape()
	seqLen := xShape[0]
	batch := xShape[1]
	inputSize := xShape[2]

	if hiddenSize == 0 {
		hiddenSize = r.Shape()[2]
	}
	H := hiddenSize
	gate3H := 3 * H

	var biasData []float32
	if len(inputs) > 3 && inputs[3] != nil {
		biasData = inputs[3].(*tensor.Dense[float32]).Data()
	}

	numDir := w.Shape()[0] // 1 = forward, 2 = bidirectional

	var hData []float32
	if len(inputs) > 5 && inputs[5] != nil {
		h := inputs[5].(*tensor.Dense[float32])
		hData = make([]float32, h.Len())
		copy(hData, h.Data())
	} else {
		hData = make([]float32, numDir*batch*H)
	}

	wData := w.Data()
	rData := r.Data()
	xData := x.Data()

	wDirStride := gate3H * inputSize // stride per direction in W
	rDirStride := gate3H * H         // stride per direction in R
	bDirStride := 2 * gate3H         // stride per direction in B

	yData := make([]float32, seqLen*numDir*batch*H)

	for dir := 0; dir < numDir; dir++ {
		wOff := dir * wDirStride
		rOff := dir * rDirStride
		bOff := 0
		if biasData != nil {
			bOff = dir * bDirStride
		}
		hOff0 := dir * batch * H // offset into hData for this direction

		// Forward: t=0..seqLen-1; Backward (dir=1): t=seqLen-1..0
		for step := 0; step < seqLen; step++ {
			t := step
			if dir == 1 {
				t = seqLen - 1 - step
			}

			for b := 0; b < batch; b++ {
				xOff := t*batch*inputSize + b*inputSize
				hIdx := hOff0 + b*H

				gates := make([]float32, gate3H)
				for g := 0; g < gate3H; g++ {
					var sum float32
					for j := 0; j < inputSize; j++ {
						sum += wData[wOff+g*inputSize+j] * xData[xOff+j]
					}
					gates[g] = sum
				}
				rGates := make([]float32, gate3H)
				for g := 0; g < gate3H; g++ {
					var sum float32
					for j := 0; j < H; j++ {
						sum += rData[rOff+g*H+j] * hData[hIdx+j]
					}
					rGates[g] = sum
				}

				if biasData != nil {
					for g := 0; g < gate3H; g++ {
						gates[g] += biasData[bOff+g]
						rGates[g] += biasData[bOff+gate3H+g]
					}
				}

				for h := 0; h < H; h++ {
					zt := sigmoid32(gates[0*H+h] + rGates[0*H+h])
					rt := sigmoid32(gates[1*H+h] + rGates[1*H+h])

					var ht float32
					if linearBeforeReset {
						ht = float32(math.Tanh(float64(gates[2*H+h] + rt*rGates[2*H+h])))
					} else {
						var sum float32
						for j := 0; j < H; j++ {
							sum += rData[rOff+(2*H+h)*H+j] * (rt * hData[hIdx+j])
						}
						if biasData != nil {
							sum += biasData[bOff+gate3H+2*H+h]
						}
						ht = float32(math.Tanh(float64(gates[2*H+h] + sum)))
					}

					hData[hIdx+h] = (1-zt)*ht + zt*hData[hIdx+h]
				}

				yOff := t*numDir*batch*H + dir*batch*H + b*H
				copy(yData[yOff:yOff+H], hData[hIdx:hIdx+H])
			}
		}
	}

	Y := tensor.NewDense[float32](tensor.Shape{seqLen, numDir, batch, H}, yData)
	Yh := tensor.NewDense[float32](tensor.Shape{numDir, batch, H}, hData)
	return []tensor.Tensor{Y, Yh}, nil
}
