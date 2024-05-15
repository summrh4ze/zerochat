package websocket

import (
	"bytes"
	"encoding/binary"
	"example/zerochat/chatProto/errors"
	"fmt"
	"net"
)

const FRAME_SIZE = 1024

const (
	maskBit              = 1 << 7
	finalBit             = 1 << 7
	opcodeMask           = 0x0F
	lenMask              = 0x7F
	continuationFrame    = 0x00
	textFrame            = 0x01
	binaryFrame          = 0x02
	connectionCloseFrame = 0x08
	pingFrame            = 0x09
	pongFrame            = 0x0A
	lenFrame             = 0x0B
)

type wsFrame struct {
	endFrame bool
	opcode   uint8
	masked   bool
	mask     []byte
	payload  []byte
}

func (f wsFrame) String() string {
	return fmt.Sprintf(
		"WS FRAME - FIN: %t, OPCODE: %0x, MASK: %t, PAYLOAD: % x",
		f.endFrame,
		f.opcode,
		f.masked,
		f.payload,
	)
}

func (f *wsFrame) write(buf []byte) error {
	// FIN bit
	if f.endFrame {
		buf[0] = buf[0] | finalBit
	}
	// RSV1, RSV2, RSV3 = 0

	// OPCODE
	buf[0] = buf[0] | f.opcode

	// MASK
	if f.masked {
		buf[1] = buf[1] | maskBit
	}

	nextIndex := 2

	// PAYLOAD LEN
	if len(f.payload) < 126 {
		buf[1] = buf[1] | uint8(len(f.payload))
	} else if len(f.payload) >= 126 && len(f.payload) < FRAME_SIZE {
		buf[1] = buf[1] | uint8(126)
		binary.BigEndian.PutUint16(buf[2:4], uint16(len(f.payload)))
		nextIndex = nextIndex + 2
	} else {
		return errors.WSFrameBuildError("payload len is greater then frame size")
	}

	// MASKING KEY
	if f.masked {
		n := copy(buf[nextIndex:nextIndex+4], f.mask)
		if n != len(f.mask) {
			return errors.WSFrameBuildError(fmt.Sprintf("mask only copied partially %d/%d", n, len(f.mask)))
		}
		nextIndex = nextIndex + 4
	}

	// PAYLOAD DATA
	n := copy(buf[nextIndex:], f.payload)
	if n != len(f.payload) {
		return errors.WSFrameBuildError(fmt.Sprintf("payload only copied partially %d/%d", n, len(f.payload)))
	}

	return nil
}

func computeHeaderLen(payloadLen int, masked bool) int {
	if payloadLen <= 125 {
		if masked {
			return 6
		}
		return 2
	} else {
		if masked {
			return 8
		}
		return 4
	}
}

func fragmentPayload(payload []byte, masked bool, frameSize int) [][]byte {
	headerLen := computeHeaderLen(len(payload), masked)

	maxPayloadSize := frameSize - headerLen
	if len(payload) <= maxPayloadSize {
		return [][]byte{payload}
	} else {
		frames := make([][]byte, 0, 10)
		completeFrames := len(payload) / maxPayloadSize
		for i := range completeFrames {
			frames = append(frames, payload[i*maxPayloadSize:(i+1)*maxPayloadSize])
		}
		if len(payload)%maxPayloadSize != 0 {
			frames = append(frames, payload[completeFrames*maxPayloadSize:])
		}
		return frames
	}
}

/* func printFrame(buf []byte, frameSize int) {
	for i := 0; i < frameSize; i += 4 {
		fmt.Printf("% x - %08b\n", buf[i:i+4], buf[i:i+4])
	}
	fmt.Println()
} */

func parseFrame(buf []byte) wsFrame {
	fin := buf[0]&finalBit == (1 << 7)

	opcode := buf[0] & opcodeMask

	mask := buf[1]&maskBit == (1 << 7)

	plen := 0
	payloadLen := buf[1] & lenMask
	nextIndex := 2

	if payloadLen == 126 {
		extendedPayloadLen := binary.BigEndian.Uint16(buf[2:4])
		plen = int(extendedPayloadLen)
		nextIndex = nextIndex + 2
	} else {
		plen = int(payloadLen)
	}

	var maskCipher []byte
	if mask {
		maskCipher = make([]byte, 4)
		copy(maskCipher, buf[nextIndex:nextIndex+4])
		nextIndex = nextIndex + 4
	}

	payload := make([]byte, plen)
	copy(payload, buf[nextIndex:nextIndex+plen])

	frame := wsFrame{
		endFrame: fin,
		opcode:   opcode,
		masked:   mask,
		mask:     maskCipher,
		payload:  payload,
	}

	fmt.Printf("RECV %v\n", frame)

	return frame
}

func ReadMessage(conn net.Conn) ([]byte, error) {
	buf := make([]byte, FRAME_SIZE)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}
	if n != FRAME_SIZE {
		return nil, errors.WSFrameReadError(fmt.Sprintf("only read %d/%d\n", n, FRAME_SIZE))
	}

	//printFrame(buf, FRAME_SIZE)
	recvFrame := parseFrame(buf)

	if recvFrame.opcode == lenFrame {
		frames := make([]wsFrame, 0)
		for {
			buf = make([]byte, FRAME_SIZE)
			_, err := conn.Read(buf)
			if err != nil {
				return nil, err
			}
			f := parseFrame(buf)
			frames = append(frames, f)
			if f.endFrame {
				break
			}
		}
		payload := make([]byte, 0, len(frames)*FRAME_SIZE)
		for _, f := range frames {
			payload = append(payload, f.payload...)
		}

		return payload, nil
	}
	return recvFrame.payload, nil
}

func CreateMessage(conn net.Conn, payload []byte, masked bool) error {
	buf := make([]byte, FRAME_SIZE)
	plen := uint64(len(payload))
	//first send a message (one frame) containing the length of the payload
	lenBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(lenBuf, plen)
	lengthFrame := wsFrame{
		endFrame: true,
		opcode:   lenFrame,
		masked:   masked,
		mask:     bytes.Repeat([]byte("a"), 4), // TODO change it
		payload:  lenBuf,
	}
	err := lengthFrame.write(buf)
	if err != nil {
		return err
	}
	fmt.Printf("SEND %v\n", lengthFrame)
	conn.Write(buf)

	fragmentedPayload := fragmentPayload([]byte(payload), masked, FRAME_SIZE)
	for i, p := range fragmentedPayload {
		buf = make([]byte, FRAME_SIZE)
		frame := wsFrame{
			endFrame: i == len(fragmentedPayload)-1,
			opcode:   textFrame,
			masked:   masked,
			mask:     bytes.Repeat([]byte("a"), 4), // TODO change it
			payload:  p,
		}
		frame.write(buf)
		fmt.Printf("SEND %v\n", frame)
		conn.Write(buf)
	}

	return nil
}
