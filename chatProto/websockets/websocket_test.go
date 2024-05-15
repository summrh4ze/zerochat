package websocket

import (
	"bytes"
	"testing"
)

func TestComputeHeaderLen(t *testing.T) {
	tests := []struct {
		plen   int
		masked bool
		want   int
	}{
		{10, true, 6},
		{10, false, 2},
		{200, true, 8},
		{200, false, 4},
	}

	for _, test := range tests {
		res := computeHeaderLen(test.plen, test.masked)
		if res != test.want {
			t.Errorf("Result is %d, want %d\n", res, test.want)
		}
	}
}

func TestFragmentPayload(t *testing.T) {
	tests := []struct {
		maxFrameSize int
		payload      []byte
		masked       bool
		want         [][]byte
	}{
		{100, bytes.Repeat([]byte("1"), 10), true, [][]byte{bytes.Repeat([]byte("1"), 10)}},
		{100, bytes.Repeat([]byte("1"), 100), true, [][]byte{bytes.Repeat([]byte("1"), 94), bytes.Repeat([]byte("1"), 6)}},
		{100, bytes.Repeat([]byte("1"), 94), true, [][]byte{bytes.Repeat([]byte("1"), 94)}},
		{100, bytes.Repeat([]byte("1"), 10), false, [][]byte{bytes.Repeat([]byte("1"), 10)}},
		{100, bytes.Repeat([]byte("1"), 100), false, [][]byte{bytes.Repeat([]byte("1"), 98), bytes.Repeat([]byte("1"), 2)}},
		{100, bytes.Repeat([]byte("1"), 98), false, [][]byte{bytes.Repeat([]byte("1"), 98)}},
		{200, bytes.Repeat([]byte("1"), 180), true, [][]byte{bytes.Repeat([]byte("1"), 180)}},
		{200, bytes.Repeat([]byte("1"), 250), true, [][]byte{bytes.Repeat([]byte("1"), 192), bytes.Repeat([]byte("1"), 58)}},
		{200, bytes.Repeat([]byte("1"), 192), true, [][]byte{bytes.Repeat([]byte("1"), 192)}},
		{200, bytes.Repeat([]byte("1"), 180), false, [][]byte{bytes.Repeat([]byte("1"), 180)}},
		{200, bytes.Repeat([]byte("1"), 250), false, [][]byte{bytes.Repeat([]byte("1"), 196), bytes.Repeat([]byte("1"), 54)}},
		{200, bytes.Repeat([]byte("1"), 196), false, [][]byte{bytes.Repeat([]byte("1"), 196)}},
		{100, bytes.Repeat([]byte("1"), 1050), true, [][]byte{
			bytes.Repeat([]byte("1"), 92),
			bytes.Repeat([]byte("1"), 92),
			bytes.Repeat([]byte("1"), 92),
			bytes.Repeat([]byte("1"), 92),
			bytes.Repeat([]byte("1"), 92),
			bytes.Repeat([]byte("1"), 92),
			bytes.Repeat([]byte("1"), 92),
			bytes.Repeat([]byte("1"), 92),
			bytes.Repeat([]byte("1"), 92),
			bytes.Repeat([]byte("1"), 92),
			bytes.Repeat([]byte("1"), 92),
			bytes.Repeat([]byte("1"), 38),
		}},
	}

	for i, test := range tests {
		res := fragmentPayload(test.payload, test.masked, test.maxFrameSize)
		if len(res) != len(test.want) {
			t.Errorf("Test %d: created %d/%d frames\n", i+1, len(res), len(test.want))
		} else {
			for j, frame := range res {
				if len(frame) != len(test.want[j]) {
					//fmt.Printf("Test %d, %#v\n", i+1, res)
					t.Errorf("Test %d: frame %d has only %d/%d bytes\n", i+1, j+1, len(frame), len(test.want[j]))
				}
			}
		}
	}
}

func TestSendMessage(t *testing.T) {
	payload := bytes.Repeat([]byte("01"), 100)
	CreateMessage(nil, payload, false)
	t.Fail()
}
