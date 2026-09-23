// Package sicp implements the Philips Serial Interface Communication Protocol
// (SICP) used by Philips professional / digital-signage displays.
//
// Reference: "The SICP Commands Document V2.05 (released 2020-11-18)".
// The wire format is unchanged since V1.88; later revisions add new
// commands and tighten valid-value ranges.
//
// Frame layout (every field is one byte):
//
//	MsgSize | Control(MonitorID) | Group | Data[0..N] | Checksum
//
//	MsgSize  = total length of the frame, i.e. 3 + 1 + len(Data)
//	         = number of bytes from MsgSize up to and including Checksum.
//	         Range 3..40 (0x03..0x28).
//	Control  = monitor ID. 1..255 = unicast, 0 = broadcast (no reply expected).
//	Group    = group ID. 0 = control by monitor ID, 1..254 = control by group.
//	Data     = command-specific payload, 0..36 bytes (0..0x24).
//	Checksum = XOR of every preceding byte (MsgSize..Data[N]).
package sicp

import (
	"errors"
	"fmt"
)

// Spec-defined limits for a SICP frame.
const (
	MinFrameSize = 5  // MsgSize + Control + Group + Data[0] + Checksum
	MaxFrameSize = 40 // 0x28
	MaxDataSize  = 36 // 0x24
)

// Broadcast is the monitor-ID value that addresses every display on the bus.
// In broadcast mode the display sends no ACK / report.
const Broadcast byte = 0x00

// Frame is a decoded SICP packet. Its fields are unexported; build one with
// NewFrame and read it through MonitorID, Group, Data, and Encode.
type Frame struct {
	monitorID byte
	group     byte
	data      []byte
}

// NewFrame builds a frame targeting monitorID (0 = broadcast) on the given
// group (0 = address by monitor ID). data must be 1..36 bytes; it is copied
// internally.
func NewFrame(monitorID, group byte, data []byte) (Frame, error) {
	if len(data) == 0 {
		return Frame{}, errors.New("sicp: frame must contain at least one data byte")
	}
	if len(data) > MaxDataSize {
		return Frame{}, fmt.Errorf("sicp: data length %d exceeds maximum %d", len(data), MaxDataSize)
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	return Frame{monitorID: monitorID, group: group, data: cp}, nil
}

// MonitorID returns the target monitor address (control byte).
func (f Frame) MonitorID() byte { return f.monitorID }

// Group returns the group ID byte.
func (f Frame) Group() byte { return f.group }

// Data returns a copy of the command + parameter bytes.
func (f Frame) Data() []byte {
	out := make([]byte, len(f.data))
	copy(out, f.data)
	return out
}

// Encode serialises the frame to its on-the-wire byte form, computing the
// MsgSize and Checksum bytes.
func (f Frame) Encode() ([]byte, error) {
	if len(f.data) == 0 {
		return nil, errors.New("sicp: frame must contain at least one data byte")
	}
	if len(f.data) > MaxDataSize {
		return nil, fmt.Errorf("sicp: data length %d exceeds maximum %d", len(f.data), MaxDataSize)
	}
	size := byte(3 + len(f.data) + 1)
	out := make([]byte, 0, size)
	out = append(out, size, f.monitorID, f.group)
	out = append(out, f.data...)
	out = append(out, checksum(out))
	return out, nil
}

// Decode parses a complete on-the-wire frame. The slice must contain exactly
// one frame; trailing bytes are reported as an error.
func Decode(buf []byte) (Frame, error) {
	if len(buf) < MinFrameSize {
		return Frame{}, fmt.Errorf("sicp: frame too short: %d bytes", len(buf))
	}
	size := int(buf[0])
	if size < MinFrameSize || size > MaxFrameSize {
		return Frame{}, fmt.Errorf("sicp: invalid MsgSize byte 0x%02X", buf[0])
	}
	if len(buf) != size {
		return Frame{}, fmt.Errorf("sicp: frame length mismatch: header says %d, got %d", size, len(buf))
	}
	if got, want := buf[size-1], checksum(buf[:size-1]); got != want {
		return Frame{}, fmt.Errorf("sicp: bad checksum: got 0x%02X, want 0x%02X", got, want)
	}
	data := make([]byte, size-4)
	copy(data, buf[3:size-1])
	return Frame{monitorID: buf[1], group: buf[2], data: data}, nil
}

// checksum XORs every byte of b, per spec §2.3.
func checksum(b []byte) byte {
	var c byte
	for _, x := range b {
		c ^= x
	}
	return c
}
