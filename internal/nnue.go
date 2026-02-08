package internal

import (
	"math"
)


const quantScale = 512.0


type Weight struct {
	Shape []int     `json:"shape"`
	Data  []float32 `json:"data"`
}


type Model struct {
	ModelType string


	EmbWeightQ []int16
	EmbDim     int


	Layer0Weight []float32
	Layer0Bias   []float32
	Layer0Out    int
	Layer2Weight []float32
	Layer2Bias   []float32
	Layer2Out    int
	Layer4Weight []float32
	Layer4Bias   []float32


	bufConcat []float32
	bufL0     []float32
	bufL2     []float32
	bufL4     []float32
}


type NNUEAccumulator struct {

	White []int32
	Black []int32


	Valid bool


	WhiteKingIdx int
	BlackKingIdx int
}


func NewAccumulator(dim int) *NNUEAccumulator {
	return &NNUEAccumulator{
		White: make([]int32, dim),
		Black: make([]int32, dim),
		Valid: false,
	}
}


func (m *Model) GetEmbDim() int {
	return m.EmbDim
}


func (a *NNUEAccumulator) Copy(dim int) *NNUEAccumulator {
	cp := &NNUEAccumulator{
		White:        make([]int32, dim),
		Black:        make([]int32, dim),
		Valid:        a.Valid,
		WhiteKingIdx: a.WhiteKingIdx,
		BlackKingIdx: a.BlackKingIdx,
	}
	copy(cp.White, a.White)
	copy(cp.Black, a.Black)
	return cp
}

func LoadModel(path string) Model {
	return Model{
		EmbDim:    256,
		bufConcat: make([]float32, 512),
		bufL0:     make([]float32, 128),
		bufL2:     make([]float32, 32),
		bufL4:     make([]float32, 1),
	}
}


func (m *Model) RefreshAccumulator(acc *NNUEAccumulator, game *Game) {
	emb := m.EmbWeightQ


	whiteKingIdx := 0
	blackKingIdx := 0
	for idx, piece := range game.Board {
		if piece != nil && piece.Type == King {
			if piece.Color == White {
				whiteKingIdx = idx
			} else {
				blackKingIdx = idx
			}
		}
	}


	for i := 0; i < 256; i += 8 {
		acc.White[i] = 0
		acc.White[i+1] = 0
		acc.White[i+2] = 0
		acc.White[i+3] = 0
		acc.White[i+4] = 0
		acc.White[i+5] = 0
		acc.White[i+6] = 0
		acc.White[i+7] = 0
		acc.Black[i] = 0
		acc.Black[i+1] = 0
		acc.Black[i+2] = 0
		acc.Black[i+3] = 0
		acc.Black[i+4] = 0
		acc.Black[i+5] = 0
		acc.Black[i+6] = 0
		acc.Black[i+7] = 0
	}

	whiteBase := whiteKingIdx * PiecesPerKing
	blackBase := blackKingIdx * PiecesPerKing


	for idx, piece := range game.Board {
		if piece == nil || piece.Type == King {
			continue
		}

		pieceType := int(piece.Type)
		if pieceType > 4 {
			continue
		}

		colorNum := 0
		if piece.Color == Black {
			colorNum = 1
		}

		pieceIdx := pieceType*(NumColors*NumSquares) + colorNum*NumSquares + idx


		whiteFeat := whiteBase + pieceIdx
		whiteRow := emb[whiteFeat*256 : (whiteFeat+1)*256]
		for i := 0; i < 256; i += 8 {
			acc.White[i] += int32(whiteRow[i])
			acc.White[i+1] += int32(whiteRow[i+1])
			acc.White[i+2] += int32(whiteRow[i+2])
			acc.White[i+3] += int32(whiteRow[i+3])
			acc.White[i+4] += int32(whiteRow[i+4])
			acc.White[i+5] += int32(whiteRow[i+5])
			acc.White[i+6] += int32(whiteRow[i+6])
			acc.White[i+7] += int32(whiteRow[i+7])
		}


		blackFeat := blackBase + pieceIdx
		blackRow := emb[blackFeat*256 : (blackFeat+1)*256]
		for i := 0; i < 256; i += 8 {
			acc.Black[i] += int32(blackRow[i])
			acc.Black[i+1] += int32(blackRow[i+1])
			acc.Black[i+2] += int32(blackRow[i+2])
			acc.Black[i+3] += int32(blackRow[i+3])
			acc.Black[i+4] += int32(blackRow[i+4])
			acc.Black[i+5] += int32(blackRow[i+5])
			acc.Black[i+6] += int32(blackRow[i+6])
			acc.Black[i+7] += int32(blackRow[i+7])
		}
	}

	acc.WhiteKingIdx = whiteKingIdx
	acc.BlackKingIdx = blackKingIdx
	acc.Valid = true
}



func (m *Model) UpdateAccumulator(acc *NNUEAccumulator, move Move, captured *Piece, game *Game) bool {

	if move.PieceType == King {
		return false
	}

	emb := m.EmbWeightQ
	whiteBase := acc.WhiteKingIdx * PiecesPerKing
	blackBase := acc.BlackKingIdx * PiecesPerKing

	pieceType := int(move.PieceType)
	colorNum := 0
	if move.PieceColor == Black {
		colorNum = 1
	}

	fromIdx := PositionToIndex(move.From)
	toIdx := PositionToIndex(move.To)


	oldPieceIdx := pieceType*(NumColors*NumSquares) + colorNum*NumSquares + fromIdx
	oldWhiteFeat := whiteBase + oldPieceIdx
	oldBlackFeat := blackBase + oldPieceIdx


	oldWhiteRow := emb[oldWhiteFeat*256 : (oldWhiteFeat+1)*256]
	oldBlackRow := emb[oldBlackFeat*256 : (oldBlackFeat+1)*256]


	for i := 0; i < 256; i += 8 {
		acc.White[i] -= int32(oldWhiteRow[i])
		acc.White[i+1] -= int32(oldWhiteRow[i+1])
		acc.White[i+2] -= int32(oldWhiteRow[i+2])
		acc.White[i+3] -= int32(oldWhiteRow[i+3])
		acc.White[i+4] -= int32(oldWhiteRow[i+4])
		acc.White[i+5] -= int32(oldWhiteRow[i+5])
		acc.White[i+6] -= int32(oldWhiteRow[i+6])
		acc.White[i+7] -= int32(oldWhiteRow[i+7])
	}

	for i := 0; i < 256; i += 8 {
		acc.Black[i] -= int32(oldBlackRow[i])
		acc.Black[i+1] -= int32(oldBlackRow[i+1])
		acc.Black[i+2] -= int32(oldBlackRow[i+2])
		acc.Black[i+3] -= int32(oldBlackRow[i+3])
		acc.Black[i+4] -= int32(oldBlackRow[i+4])
		acc.Black[i+5] -= int32(oldBlackRow[i+5])
		acc.Black[i+6] -= int32(oldBlackRow[i+6])
		acc.Black[i+7] -= int32(oldBlackRow[i+7])
	}


	newPieceType := pieceType
	if move.IsPromotion && move.PromotionType != nil {
		newPieceType = int(*move.PromotionType)
	}

	newPieceIdx := newPieceType*(NumColors*NumSquares) + colorNum*NumSquares + toIdx
	newWhiteFeat := whiteBase + newPieceIdx
	newBlackFeat := blackBase + newPieceIdx

	newWhiteRow := emb[newWhiteFeat*256 : (newWhiteFeat+1)*256]
	newBlackRow := emb[newBlackFeat*256 : (newBlackFeat+1)*256]


	for i := 0; i < 256; i += 8 {
		acc.White[i] += int32(newWhiteRow[i])
		acc.White[i+1] += int32(newWhiteRow[i+1])
		acc.White[i+2] += int32(newWhiteRow[i+2])
		acc.White[i+3] += int32(newWhiteRow[i+3])
		acc.White[i+4] += int32(newWhiteRow[i+4])
		acc.White[i+5] += int32(newWhiteRow[i+5])
		acc.White[i+6] += int32(newWhiteRow[i+6])
		acc.White[i+7] += int32(newWhiteRow[i+7])
	}

	for i := 0; i < 256; i += 8 {
		acc.Black[i] += int32(newBlackRow[i])
		acc.Black[i+1] += int32(newBlackRow[i+1])
		acc.Black[i+2] += int32(newBlackRow[i+2])
		acc.Black[i+3] += int32(newBlackRow[i+3])
		acc.Black[i+4] += int32(newBlackRow[i+4])
		acc.Black[i+5] += int32(newBlackRow[i+5])
		acc.Black[i+6] += int32(newBlackRow[i+6])
		acc.Black[i+7] += int32(newBlackRow[i+7])
	}


	if captured != nil {
		capType := int(captured.Type)
		capColor := 0
		if captured.Color == Black {
			capColor = 1
		}
		capIdx := PositionToIndex(captured.Position)

		capPieceIdx := capType*(NumColors*NumSquares) + capColor*NumSquares + capIdx
		capWhiteFeat := whiteBase + capPieceIdx
		capBlackFeat := blackBase + capPieceIdx

		capWhiteRow := emb[capWhiteFeat*256 : (capWhiteFeat+1)*256]
		capBlackRow := emb[capBlackFeat*256 : (capBlackFeat+1)*256]

		for i := 0; i < 256; i += 8 {
			acc.White[i] -= int32(capWhiteRow[i])
			acc.White[i+1] -= int32(capWhiteRow[i+1])
			acc.White[i+2] -= int32(capWhiteRow[i+2])
			acc.White[i+3] -= int32(capWhiteRow[i+3])
			acc.White[i+4] -= int32(capWhiteRow[i+4])
			acc.White[i+5] -= int32(capWhiteRow[i+5])
			acc.White[i+6] -= int32(capWhiteRow[i+6])
			acc.White[i+7] -= int32(capWhiteRow[i+7])
		}
		for i := 0; i < 256; i += 8 {
			acc.Black[i] -= int32(capBlackRow[i])
			acc.Black[i+1] -= int32(capBlackRow[i+1])
			acc.Black[i+2] -= int32(capBlackRow[i+2])
			acc.Black[i+3] -= int32(capBlackRow[i+3])
			acc.Black[i+4] -= int32(capBlackRow[i+4])
			acc.Black[i+5] -= int32(capBlackRow[i+5])
			acc.Black[i+6] -= int32(capBlackRow[i+6])
			acc.Black[i+7] -= int32(capBlackRow[i+7])
		}
	}

	return true
}


func (m *Model) PredictWithAccumulator(acc *NNUEAccumulator, turn Color) float32 {
	invScale := float32(1.0 / quantScale)
	concat := m.bufConcat



	if turn == White {
		for i := 0; i < 256; i += 8 {
			concat[i] = float32(acc.White[i]) * invScale
			concat[i+1] = float32(acc.White[i+1]) * invScale
			concat[i+2] = float32(acc.White[i+2]) * invScale
			concat[i+3] = float32(acc.White[i+3]) * invScale
			concat[i+4] = float32(acc.White[i+4]) * invScale
			concat[i+5] = float32(acc.White[i+5]) * invScale
			concat[i+6] = float32(acc.White[i+6]) * invScale
			concat[i+7] = float32(acc.White[i+7]) * invScale
			concat[256+i] = float32(acc.Black[i]) * invScale
			concat[256+i+1] = float32(acc.Black[i+1]) * invScale
			concat[256+i+2] = float32(acc.Black[i+2]) * invScale
			concat[256+i+3] = float32(acc.Black[i+3]) * invScale
			concat[256+i+4] = float32(acc.Black[i+4]) * invScale
			concat[256+i+5] = float32(acc.Black[i+5]) * invScale
			concat[256+i+6] = float32(acc.Black[i+6]) * invScale
			concat[256+i+7] = float32(acc.Black[i+7]) * invScale
		}
	} else {
		for i := 0; i < 256; i += 8 {
			concat[i] = float32(acc.Black[i]) * invScale
			concat[i+1] = float32(acc.Black[i+1]) * invScale
			concat[i+2] = float32(acc.Black[i+2]) * invScale
			concat[i+3] = float32(acc.Black[i+3]) * invScale
			concat[i+4] = float32(acc.Black[i+4]) * invScale
			concat[i+5] = float32(acc.Black[i+5]) * invScale
			concat[i+6] = float32(acc.Black[i+6]) * invScale
			concat[i+7] = float32(acc.Black[i+7]) * invScale
			concat[256+i] = float32(acc.White[i]) * invScale
			concat[256+i+1] = float32(acc.White[i+1]) * invScale
			concat[256+i+2] = float32(acc.White[i+2]) * invScale
			concat[256+i+3] = float32(acc.White[i+3]) * invScale
			concat[256+i+4] = float32(acc.White[i+4]) * invScale
			concat[256+i+5] = float32(acc.White[i+5]) * invScale
			concat[256+i+6] = float32(acc.White[i+6]) * invScale
			concat[256+i+7] = float32(acc.White[i+7]) * invScale
		}
	}


	m.linear512to128CReLU(concat, m.Layer0Weight, m.Layer0Bias, m.bufL0)


	m.linear128to32CReLU(m.bufL0, m.Layer2Weight, m.Layer2Bias, m.bufL2)


	m.linear32to1Sigmoid(m.bufL2, m.Layer4Weight, m.Layer4Bias, m.bufL4)

	return m.bufL4[0]
}


func (m *Model) Predict(ourFeatures, theirFeatures []int) float32 {
	dim := m.EmbDim
	emb := m.EmbWeightQ


	buf0 := make([]int32, dim)
	buf1 := make([]int32, dim)


	for _, feat := range ourFeatures {
		offset := feat * dim
		for i := 0; i < dim; i++ {
			buf0[i] += int32(emb[offset+i])
		}
	}


	for _, feat := range theirFeatures {
		offset := feat * dim
		for i := 0; i < dim; i++ {
			buf1[i] += int32(emb[offset+i])
		}
	}


	invScale := float32(1.0 / quantScale)
	concat := m.bufConcat
	for i := 0; i < dim; i++ {
		concat[i] = float32(buf0[i]) * invScale
		concat[dim+i] = float32(buf1[i]) * invScale
	}


	m.linearCReLU(concat, m.Layer0Weight, m.Layer0Bias, m.bufL0, 512, m.Layer0Out)


	m.linearCReLU(m.bufL0, m.Layer2Weight, m.Layer2Bias, m.bufL2, m.Layer0Out, m.Layer2Out)


	m.linearSigmoid(m.bufL2, m.Layer4Weight, m.Layer4Bias, m.bufL4, m.Layer2Out, 1)

	return m.bufL4[0]
}



func (m *Model) linear512to128CReLU(x, weight, bias, out []float32) {

	for i := 0; i < 128; i += 4 {
		sum0 := bias[i]
		sum1 := bias[i+1]
		sum2 := bias[i+2]
		sum3 := bias[i+3]

		row0 := weight[i*512 : (i+1)*512]
		row1 := weight[(i+1)*512 : (i+2)*512]
		row2 := weight[(i+2)*512 : (i+3)*512]
		row3 := weight[(i+3)*512 : (i+4)*512]


		for j := 0; j < 512; j += 8 {
			xj0 := x[j]
			xj1 := x[j+1]
			xj2 := x[j+2]
			xj3 := x[j+3]
			xj4 := x[j+4]
			xj5 := x[j+5]
			xj6 := x[j+6]
			xj7 := x[j+7]

			sum0 += row0[j]*xj0 + row0[j+1]*xj1 + row0[j+2]*xj2 + row0[j+3]*xj3
			sum0 += row0[j+4]*xj4 + row0[j+5]*xj5 + row0[j+6]*xj6 + row0[j+7]*xj7

			sum1 += row1[j]*xj0 + row1[j+1]*xj1 + row1[j+2]*xj2 + row1[j+3]*xj3
			sum1 += row1[j+4]*xj4 + row1[j+5]*xj5 + row1[j+6]*xj6 + row1[j+7]*xj7

			sum2 += row2[j]*xj0 + row2[j+1]*xj1 + row2[j+2]*xj2 + row2[j+3]*xj3
			sum2 += row2[j+4]*xj4 + row2[j+5]*xj5 + row2[j+6]*xj6 + row2[j+7]*xj7

			sum3 += row3[j]*xj0 + row3[j+1]*xj1 + row3[j+2]*xj2 + row3[j+3]*xj3
			sum3 += row3[j+4]*xj4 + row3[j+5]*xj5 + row3[j+6]*xj6 + row3[j+7]*xj7
		}


		if sum0 < 0 {
			sum0 = 0
		} else if sum0 > 1 {
			sum0 = 1
		}
		if sum1 < 0 {
			sum1 = 0
		} else if sum1 > 1 {
			sum1 = 1
		}
		if sum2 < 0 {
			sum2 = 0
		} else if sum2 > 1 {
			sum2 = 1
		}
		if sum3 < 0 {
			sum3 = 0
		} else if sum3 > 1 {
			sum3 = 1
		}

		out[i] = sum0
		out[i+1] = sum1
		out[i+2] = sum2
		out[i+3] = sum3
	}
}



func (m *Model) linear128to32CReLU(x, weight, bias, out []float32) {
	for i := 0; i < 32; i += 4 {
		sum0 := bias[i]
		sum1 := bias[i+1]
		sum2 := bias[i+2]
		sum3 := bias[i+3]

		row0 := weight[i*128 : (i+1)*128]
		row1 := weight[(i+1)*128 : (i+2)*128]
		row2 := weight[(i+2)*128 : (i+3)*128]
		row3 := weight[(i+3)*128 : (i+4)*128]

		for j := 0; j < 128; j += 8 {
			xj0 := x[j]
			xj1 := x[j+1]
			xj2 := x[j+2]
			xj3 := x[j+3]
			xj4 := x[j+4]
			xj5 := x[j+5]
			xj6 := x[j+6]
			xj7 := x[j+7]

			sum0 += row0[j]*xj0 + row0[j+1]*xj1 + row0[j+2]*xj2 + row0[j+3]*xj3
			sum0 += row0[j+4]*xj4 + row0[j+5]*xj5 + row0[j+6]*xj6 + row0[j+7]*xj7

			sum1 += row1[j]*xj0 + row1[j+1]*xj1 + row1[j+2]*xj2 + row1[j+3]*xj3
			sum1 += row1[j+4]*xj4 + row1[j+5]*xj5 + row1[j+6]*xj6 + row1[j+7]*xj7

			sum2 += row2[j]*xj0 + row2[j+1]*xj1 + row2[j+2]*xj2 + row2[j+3]*xj3
			sum2 += row2[j+4]*xj4 + row2[j+5]*xj5 + row2[j+6]*xj6 + row2[j+7]*xj7

			sum3 += row3[j]*xj0 + row3[j+1]*xj1 + row3[j+2]*xj2 + row3[j+3]*xj3
			sum3 += row3[j+4]*xj4 + row3[j+5]*xj5 + row3[j+6]*xj6 + row3[j+7]*xj7
		}


		if sum0 < 0 {
			sum0 = 0
		} else if sum0 > 1 {
			sum0 = 1
		}
		if sum1 < 0 {
			sum1 = 0
		} else if sum1 > 1 {
			sum1 = 1
		}
		if sum2 < 0 {
			sum2 = 0
		} else if sum2 > 1 {
			sum2 = 1
		}
		if sum3 < 0 {
			sum3 = 0
		} else if sum3 > 1 {
			sum3 = 1
		}

		out[i] = sum0
		out[i+1] = sum1
		out[i+2] = sum2
		out[i+3] = sum3
	}
}


func (m *Model) linear32to1Sigmoid(x, weight, bias, out []float32) {
	sum := bias[0]

	sum += weight[0] * x[0]
	sum += weight[1] * x[1]
	sum += weight[2] * x[2]
	sum += weight[3] * x[3]
	sum += weight[4] * x[4]
	sum += weight[5] * x[5]
	sum += weight[6] * x[6]
	sum += weight[7] * x[7]
	sum += weight[8] * x[8]
	sum += weight[9] * x[9]
	sum += weight[10] * x[10]
	sum += weight[11] * x[11]
	sum += weight[12] * x[12]
	sum += weight[13] * x[13]
	sum += weight[14] * x[14]
	sum += weight[15] * x[15]
	sum += weight[16] * x[16]
	sum += weight[17] * x[17]
	sum += weight[18] * x[18]
	sum += weight[19] * x[19]
	sum += weight[20] * x[20]
	sum += weight[21] * x[21]
	sum += weight[22] * x[22]
	sum += weight[23] * x[23]
	sum += weight[24] * x[24]
	sum += weight[25] * x[25]
	sum += weight[26] * x[26]
	sum += weight[27] * x[27]
	sum += weight[28] * x[28]
	sum += weight[29] * x[29]
	sum += weight[30] * x[30]
	sum += weight[31] * x[31]

	out[0] = fastSigmoid(sum)
}



func fastSigmoid(x float32) float32 {

	if x < -8 {
		return 0.0003
	}
	if x > 8 {
		return 0.9997
	}


	ax := x
	if ax < 0 {
		ax = -ax
	}
	return 0.5 + x/(4.0+2.0*ax+x*x)
}



func (m *Model) linearCReLU(x, weight, bias, out []float32, inDim, outDim int) {
	for i := 0; i < outDim; i++ {
		sum := bias[i]
		rowOffset := i * inDim


		j := 0
		for ; j <= inDim-8; j += 8 {
			sum += weight[rowOffset+j] * x[j]
			sum += weight[rowOffset+j+1] * x[j+1]
			sum += weight[rowOffset+j+2] * x[j+2]
			sum += weight[rowOffset+j+3] * x[j+3]
			sum += weight[rowOffset+j+4] * x[j+4]
			sum += weight[rowOffset+j+5] * x[j+5]
			sum += weight[rowOffset+j+6] * x[j+6]
			sum += weight[rowOffset+j+7] * x[j+7]
		}

		for ; j < inDim; j++ {
			sum += weight[rowOffset+j] * x[j]
		}


		if sum < 0 {
			sum = 0
		} else if sum > 1 {
			sum = 1
		}
		out[i] = sum
	}
}


func (m *Model) linearSigmoid(x, weight, bias, out []float32, inDim, outDim int) {
	for i := 0; i < outDim; i++ {
		sum := bias[i]
		rowOffset := i * inDim
		for j := 0; j < inDim; j++ {
			sum += weight[rowOffset+j] * x[j]
		}
		out[i] = float32(1.0 / (1.0 + math.Exp(float64(-sum))))
	}
}


const (
	NumSquares    = 91
	NumPieceTypes = 5
	NumColors     = 2
	PiecesPerKing = NumPieceTypes * NumColors * NumSquares
)


func BoardToHalfKPFeatures(game *Game, stmBuf, nstmBuf []int) ([]int, []int) {
	whiteKingIdx := 0
	blackKingIdx := 0
	for idx, piece := range game.Board {
		if piece != nil && piece.Type == King {
			if piece.Color == White {
				whiteKingIdx = idx
			} else {
				blackKingIdx = idx
			}
		}
	}

	stmBuf = stmBuf[:0]
	nstmBuf = nstmBuf[:0]

	whiteBase := whiteKingIdx * PiecesPerKing
	blackBase := blackKingIdx * PiecesPerKing

	for idx, piece := range game.Board {
		if piece == nil || piece.Type == King {
			continue
		}

		pieceType := int(piece.Type)
		if pieceType > 4 {
			continue
		}

		colorNum := 0
		if piece.Color == Black {
			colorNum = 1
		}

		pieceIdx := pieceType*(NumColors*NumSquares) + colorNum*NumSquares + idx

		whiteFeat := whiteBase + pieceIdx
		blackFeat := blackBase + pieceIdx

		if game.Turn == White {
			stmBuf = append(stmBuf, whiteFeat)
			nstmBuf = append(nstmBuf, blackFeat)
		} else {
			stmBuf = append(stmBuf, blackFeat)
			nstmBuf = append(nstmBuf, whiteFeat)
		}
	}

	return stmBuf, nstmBuf
}


var (
	featureBufSTM  = make([]int, 0, 32)
	featureBufNSTM = make([]int, 0, 32)
)


func BoardToHalfKPFeaturesAlloc(game *Game) ([]int, []int) {
	return BoardToHalfKPFeatures(game, featureBufSTM, featureBufNSTM)
}
