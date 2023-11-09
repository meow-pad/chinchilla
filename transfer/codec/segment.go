package codec

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/meow-pad/chinchilla/utils/codec"
	"github.com/meow-pad/persian/frame/pnet"
	"github.com/meow-pad/persian/utils/numeric"
	"math"
)

const (
	segmentHeadSize = 1 + 2 + 2 + 2
	maxSegmentNum   = math.MaxUint16
)

func MessageSegmentation(largeMsg []byte, maxLen int) (out []byte, err error) {
	msgLen := len(largeMsg)
	segmentNum := msgLen/maxLen + 1
	totalLen := msgLen + segmentNum*segmentHeadSize
	segmentNum = totalLen / maxLen
	if totalLen%maxLen != 0 {
		segmentNum += 1
	}
	if segmentNum > maxSegmentNum {
		return nil, pnet.ErrMessageTooLarge
	}
	segmentFrameSize := msgLen / segmentNum
	if (segmentFrameSize + segmentHeadSize) > maxLen {
		// ????
		return nil, fmt.Errorf("invalid segment frame size:%d, maxLen:%d", segmentFrameSize, maxLen)
	}
	left := largeMsg
	leftLen := len(left)
	for i := 0; leftLen > 0; i++ {
		frameLen := numeric.Min[int](leftLen, segmentFrameSize)
		sMsg := &SegmentMsg{
			Amount: uint16(segmentNum),
			Seq:    uint16(i),
			Frame:  left[:frameLen],
		}
		var sOut []byte
		sOut, err = encodeSegmentMsg(MessageCodecByteOrder, sMsg)
		if err != nil {
			return
		}
		out = append(out, sOut...)
		left = left[frameLen:]
		leftLen = len(left)
	}
	return
}

func encodeSegmentMsg(byteOrder binary.ByteOrder, msg *SegmentMsg) ([]byte, error) {
	buf := make([]byte, len(msg.Frame)+2+2+1)
	buf[0] = TypeSegment
	byteOrder.PutUint16(buf[1:], msg.Amount)
	byteOrder.PutUint16(buf[3:], msg.Seq)
	copy(buf[5:], msg.Frame)
	return buf, nil
}

func decodeSegmentMsg(byteOrder binary.ByteOrder, buf []byte) (any, error) {
	req := &SegmentMsg{}
	err := error(nil)
	if req.Amount, buf, err = codec.ReadUint16(byteOrder, buf); err != nil {
		return nil, err
	}
	if req.Seq, buf, err = codec.ReadUint16(byteOrder, buf); err != nil {
		return nil, err
	}
	req.Frame = bytes.Clone(buf)
	return req, nil
}
