package codec

import (
	"encoding/binary"
	"github.com/meow-pad/persian/frame/pnet"
	"io"
	"math"
)

func Uint64ArrayLen(arr []uint64) int {
	arrLen := len(arr)
	return 8*arrLen + 2
}

func ReadUint64Array(byteOrder binary.ByteOrder, buf []byte) (result []uint64, left []byte, err error) {
	if len(buf) < 2 {
		return nil, nil, io.ErrShortBuffer
	}
	left = buf[2:]
	arrLen := int(byteOrder.Uint16(buf))
	if arrLen*8 > len(left) {
		return nil, nil, io.ErrShortBuffer
	}
	for i := 0; i < arrLen; i++ {
		value := byteOrder.Uint64(left)
		result = append(result, value)
		left = left[8:]
	}
	return
}

func ReadInt16(byteOrder binary.ByteOrder, buf []byte) (result int16, left []byte, err error) {
	var ur uint16
	ur, left, err = ReadUint16(byteOrder, buf)
	if err != nil {
		return
	}
	result = int16(ur)
	return
}

func WriteInt16(byteOrder binary.ByteOrder, data int16, buf []byte) (left []byte, err error) {
	return WriteUint16(byteOrder, uint16(data), buf)
}

func ReadUint16(byteOrder binary.ByteOrder, buf []byte) (result uint16, left []byte, err error) {
	if len(buf) < 2 {
		return 0, nil, io.ErrShortBuffer
	}
	result = byteOrder.Uint16(buf)
	left = buf[2:]
	return
}

func WriteUint16(byteOrder binary.ByteOrder, data uint16, buf []byte) (left []byte, err error) {
	lenBuf := len(buf)
	if lenBuf < 2 {
		return nil, io.ErrShortWrite
	}
	byteOrder.PutUint16(buf, data)
	return buf[2:], nil
}

func ReadUint32(byteOrder binary.ByteOrder, buf []byte) (result uint32, left []byte, err error) {
	if len(buf) < 4 {
		return 0, nil, io.ErrShortBuffer
	}
	result = byteOrder.Uint32(buf)
	left = buf[4:]
	return
}

func WriteUint32(byteOrder binary.ByteOrder, data uint32, buf []byte) (left []byte, err error) {
	lenBuf := len(buf)
	if lenBuf < 4 {
		return nil, io.ErrShortWrite
	}
	byteOrder.PutUint32(buf, data)
	return buf[4:], nil
}

func ReadUint64(byteOrder binary.ByteOrder, buf []byte) (result uint64, left []byte, err error) {
	if len(buf) < 8 {
		return 0, nil, io.ErrShortBuffer
	}
	result = byteOrder.Uint64(buf)
	left = buf[8:]
	return
}

func WriteUint64Array(byteOrder binary.ByteOrder, data []uint64, buf []byte) (left []byte, err error) {
	lenBuf := len(buf)
	if lenBuf < 2 {
		return nil, io.ErrShortWrite
	}
	arrLen := len(data)
	if arrLen >= math.MaxUint16 {
		return nil, pnet.ErrMessageTooLarge
	}
	byteOrder.PutUint16(buf, uint16(arrLen))
	left = buf[2:]
	if len(left) < arrLen*8 {
		return nil, io.ErrShortWrite
	}
	for _, value := range data {
		byteOrder.PutUint64(left, value)
		left = left[8:]
	}
	return
}

func WriteUint64(byteOrder binary.ByteOrder, data uint64, buf []byte) (left []byte, err error) {
	lenBuf := len(buf)
	if lenBuf < 8 {
		return nil, io.ErrShortWrite
	}
	byteOrder.PutUint64(buf, data)
	return buf[8:], nil
}

func ReadString(byteOrder binary.ByteOrder, buf []byte) (result string, left []byte, err error) {
	lenBuf := len(buf)
	offset := 0
	if lenBuf < 2 {
		return "", nil, io.ErrShortBuffer
	}
	sLen := int(byteOrder.Uint16(buf))
	offset += 2
	maxOffset := offset + sLen
	if lenBuf < maxOffset {
		return "", nil, io.ErrShortBuffer
	}
	return string(buf[offset:maxOffset]), buf[maxOffset:], nil
}

func WriteString(byteOrder binary.ByteOrder, data string, buf []byte) (left []byte, err error) {
	lenBuf := len(buf)
	lenData := len(data)
	if lenBuf < 2+lenData {
		return nil, io.ErrShortWrite
	}
	if lenData > math.MaxUint16 {
		return nil, pnet.ErrMessageTooLarge
	}
	byteOrder.PutUint16(buf, uint16(lenData))
	copy(buf[2:], data)
	return buf[2+lenData:], nil
}
