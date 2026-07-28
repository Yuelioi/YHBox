package installed

import (
	"bytes"
	"image"
	"image/png"
)

func encodeCapturePNG(frame image.Image) ([]byte, error) {
	var encoded bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := encoder.Encode(&encoded, frame); err != nil {
		return nil, err
	}
	return encoded.Bytes(), nil
}
