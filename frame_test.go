package sicp

import (
	"bytes"
	"testing"
)

// Test vectors from "The SICP Commands Document V1.88".

func mustEncode(t *testing.T, monitor, group byte, data ...byte) []byte {
	t.Helper()
	f, err := NewFrame(monitor, group, data)
	if err != nil {
		t.Fatalf("NewFrame: %v", err)
	}
	wire, err := f.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	return wire
}

func TestEncode_PowerGet(t *testing.T) {
	// §4.1.1 Get power state, monitor 0x01: 05 01 00 19 1D
	got := mustEncode(t, 0x01, 0x00, CmdGetPower)
	want := []byte{0x05, 0x01, 0x00, 0x19, 0x1D}
	if !bytes.Equal(got, want) {
		t.Fatalf("encode mismatch:\n got %X\nwant %X", got, want)
	}
}

func TestEncode_PowerSetOff(t *testing.T) {
	// §4.1.3 Set power off, monitor 0x01: 06 01 00 18 01 1E
	got := mustEncode(t, 0x01, 0x00, CmdSetPower, PowerOff)
	want := []byte{0x06, 0x01, 0x00, 0x18, 0x01, 0x1E}
	if !bytes.Equal(got, want) {
		t.Fatalf("encode mismatch:\n got %X\nwant %X", got, want)
	}
}

func TestEncode_VolumeSet(t *testing.T) {
	// §7.1.3 Speaker=0x16, AudioOut=0x32, monitor 0x01:
	// 07 01 00 44 16 32 66
	got := mustEncode(t, 0x01, 0x00, CmdSetVolume, 0x16, 0x32)
	want := []byte{0x07, 0x01, 0x00, 0x44, 0x16, 0x32, 0x66}
	if !bytes.Equal(got, want) {
		t.Fatalf("encode mismatch:\n got %X\nwant %X", got, want)
	}
}

func TestEncode_VideoSet(t *testing.T) {
	// §6.1.3 All video params 0x37, gamma 0x03, monitor 0x01:
	// 0C 01 00 32 37 37 37 37 37 37 03 3C
	got := mustEncode(t, 0x01, 0x00, CmdSetVideo, 0x37, 0x37, 0x37, 0x37, 0x37, 0x37, 0x03)
	want := []byte{0x0C, 0x01, 0x00, 0x32, 0x37, 0x37, 0x37, 0x37, 0x37, 0x37, 0x03, 0x3C}
	if !bytes.Equal(got, want) {
		t.Fatalf("encode mismatch:\n got %X\nwant %X", got, want)
	}
}

func TestEncode_PlatformGet(t *testing.T) {
	// §3.2.1 Get SICP version, monitor 0x01: 06 01 00 A2 00 A5
	got := mustEncode(t, 0x01, 0x00, CmdGetPlatform, 0x00)
	want := []byte{0x06, 0x01, 0x00, 0xA2, 0x00, 0xA5}
	if !bytes.Equal(got, want) {
		t.Fatalf("encode mismatch:\n got %X\nwant %X", got, want)
	}
}

func TestDecode_PowerOnReport(t *testing.T) {
	// §4.1.2 Power report on: 06 01 00 19 02 1C
	wire := []byte{0x06, 0x01, 0x00, 0x19, 0x02, 0x1C}
	f, err := Decode(wire)
	if err != nil {
		t.Fatal(err)
	}
	if f.MonitorID() != 0x01 || f.Group() != 0x00 {
		t.Fatalf("header mismatch: monitor=%#x group=%#x", f.MonitorID(), f.Group())
	}
	if !bytes.Equal(f.Data(), []byte{0x19, 0x02}) {
		t.Fatalf("data mismatch: %X", f.Data())
	}
}

func TestFrame_DataIsCopy(t *testing.T) {
	// Mutating the slice returned by Data() must not affect the frame.
	f, err := NewFrame(0x01, 0x00, []byte{0x19, 0x02})
	if err != nil {
		t.Fatal(err)
	}
	d := f.Data()
	d[0] = 0xFF
	if f.Data()[0] != 0x19 {
		t.Fatalf("Data() returned a live reference, frame mutated to %X", f.Data())
	}
}

func TestFrame_InputIsCopied(t *testing.T) {
	// Mutating the caller's slice after NewFrame must not affect the frame.
	src := []byte{0x19, 0x02}
	f, err := NewFrame(0x01, 0x00, src)
	if err != nil {
		t.Fatal(err)
	}
	src[0] = 0xFF
	if f.Data()[0] != 0x19 {
		t.Fatalf("NewFrame retained caller slice, frame mutated to %X", f.Data())
	}
}

func TestDecode_BadChecksum(t *testing.T) {
	wire := []byte{0x06, 0x01, 0x00, 0x19, 0x02, 0xFF}
	if _, err := Decode(wire); err == nil {
		t.Fatal("expected checksum error")
	}
}

func TestDecode_LengthMismatch(t *testing.T) {
	// Header says 7, only 6 bytes given.
	wire := []byte{0x07, 0x01, 0x00, 0x19, 0x02, 0x1C}
	if _, err := Decode(wire); err == nil {
		t.Fatal("expected length error")
	}
}

func TestEncode_MaxData(t *testing.T) {
	d := make([]byte, MaxDataSize)
	for i := range d {
		d[i] = 0xAA
	}
	f, err := NewFrame(1, 0, d)
	if err != nil {
		t.Fatal(err)
	}
	wire, err := f.Encode()
	if err != nil {
		t.Fatal(err)
	}
	if len(wire) != MaxFrameSize {
		t.Fatalf("max frame should be %d bytes, got %d", MaxFrameSize, len(wire))
	}
	if _, err := Decode(wire); err != nil {
		t.Fatalf("roundtrip decode: %v", err)
	}
}

func TestNewFrame_Overflow(t *testing.T) {
	d := make([]byte, MaxDataSize+1)
	if _, err := NewFrame(1, 0, d); err == nil {
		t.Fatal("expected overflow error")
	}
}

func TestNewFrame_EmptyData(t *testing.T) {
	if _, err := NewFrame(1, 0, nil); err == nil {
		t.Fatal("expected empty-data error")
	}
}

func TestSourceFromName(t *testing.T) {
	cases := map[string]byte{
		"hdmi":        SrcHDMI,
		"DVI-D":       SrcDVID,
		"VGA":         SrcVGA,
		"hdmi4":       SrcHDMI4,
		"MediaPlayer": SrcMediaPlayer,
		"pdfplayer":   SrcPDFPlayer,
		"cmndplayweb": SrcCMNDPlayWeb,
		"iwb":         SrcIWB,
	}
	for name, want := range cases {
		got, ok := SourceFromName(name)
		if !ok || got != want {
			t.Errorf("SourceFromName(%q) = (%#x, %v); want (%#x, true)", name, got, ok, want)
		}
	}
	if _, ok := SourceFromName("nope"); ok {
		t.Error("expected unknown source to fail")
	}
}

func TestVideoParams_Accessors(t *testing.T) {
	p := NewVideoParams(10, 20, 30, 40, 50, 60, 0x03)
	if p.Brightness() != 10 || p.Color() != 20 || p.Contrast() != 30 ||
		p.Sharpness() != 40 || p.Tint() != 50 || p.BlackLevel() != 60 || p.Gamma() != 0x03 {
		t.Fatalf("accessors returned wrong values: %+v", p)
	}
}
