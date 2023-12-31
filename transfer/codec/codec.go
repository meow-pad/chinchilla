package codec

import (
	"encoding/binary"
	"github.com/meow-pad/persian/frame/pnet/message"
	"github.com/meow-pad/persian/frame/pnet/tcp/codec"
	"math"
)

var (
	MessageCodecByteOrder = binary.LittleEndian
)

func NewCodec(msgCodec message.Codec, messageWarningSize int) (codec.Codec, error) {
	opts := []codec.Option[*codec.LengthOptions]{
		codec.WithMessageCodec[*codec.LengthOptions](msgCodec),
		codec.WithByteOrder(MessageCodecByteOrder),
		codec.WithLengthSize(2),
		codec.WithMaxDecodedLength[*codec.LengthOptions](math.MaxInt16),
		codec.WithMaxEncodedLength[*codec.LengthOptions](math.MaxInt16),
		codec.WithWarningEncodedLength[*codec.LengthOptions](messageWarningSize),
		codec.WithEncodeLargeMessage(MessageSegmentation),
	}
	return codec.NewLengthFieldCodec(opts...)
}
