package sicp

import (
	"bytes"
	"errors"
	"testing"
)

// fakeTransport is a scriptable Transport for client-level tests. It records
// every Send and replays Receive() answers from a preloaded queue.
type fakeTransport struct {
	sent      [][]byte
	replies   [][]byte
	replyIdx  int
	closed    bool
	sendErr   error
	receiveOn bool // true once at least one Send has occurred (to mimic req/resp pairing)
}

func (f *fakeTransport) Send(frame []byte) error {
	if f.sendErr != nil {
		return f.sendErr
	}
	cp := make([]byte, len(frame))
	copy(cp, frame)
	f.sent = append(f.sent, cp)
	f.receiveOn = true
	return nil
}

func (f *fakeTransport) Receive() ([]byte, error) {
	if f.replyIdx >= len(f.replies) {
		return nil, errors.New("fakeTransport: no reply queued")
	}
	r := f.replies[f.replyIdx]
	f.replyIdx++
	return r, nil
}

func (f *fakeTransport) Close() error { f.closed = true; return nil }

// frame builds a wire frame for the given monitor/group/data, panicking on
// invalid input — handy for table-driven test fixtures.
func frame(t *testing.T, monitor, group byte, data ...byte) []byte {
	t.Helper()
	f, err := NewFrame(monitor, group, data)
	if err != nil {
		t.Fatal(err)
	}
	wire, err := f.Encode()
	if err != nil {
		t.Fatal(err)
	}
	return wire
}

func TestClient_SetMute(t *testing.T) {
	ft := &fakeTransport{replies: [][]byte{
		// Generic ACK report: 06 01 00 00 06 01
		{0x06, 0x01, 0x00, 0x00, 0x06, 0x01},
	}}
	c := NewClient(ft, 1, 0)
	if err := c.SetMute(true); err != nil {
		t.Fatalf("SetMute: %v", err)
	}
	want := frame(t, 1, 0, CmdSetMute, MuteOn)
	if !bytes.Equal(ft.sent[0], want) {
		t.Fatalf("sent frame:\n got %X\nwant %X", ft.sent[0], want)
	}
}

func TestClient_Mute_Get(t *testing.T) {
	// Mute report: 06 01 00 46 01 40 (mute on)
	ft := &fakeTransport{replies: [][]byte{frame(t, 1, 0, CmdGetMute, MuteOn)}}
	c := NewClient(ft, 1, 0)
	on, err := c.Mute()
	if err != nil {
		t.Fatalf("Mute: %v", err)
	}
	if !on {
		t.Fatal("Mute() = false; want true")
	}
}

func TestClient_Backlight_Roundtrip(t *testing.T) {
	// Spec example: 06 01 00 71 00 76 (backlight on)  /  06 01 00 71 01 77 (off)
	ft := &fakeTransport{replies: [][]byte{
		{0x06, 0x01, 0x00, 0x71, 0x00, 0x76},
		{0x06, 0x01, 0x00, 0x71, 0x01, 0x77},
	}}
	c := NewClient(ft, 1, 0)
	if on, err := c.Backlight(); err != nil || !on {
		t.Fatalf("Backlight #1 = (%v, %v); want (true, nil)", on, err)
	}
	if on, err := c.Backlight(); err != nil || on {
		t.Fatalf("Backlight #2 = (%v, %v); want (false, nil)", on, err)
	}
}

func TestClient_SetBacklight_OnOffWiring(t *testing.T) {
	// Make sure the API hides the inverted convention (0x00 == on in the spec).
	ft := &fakeTransport{replies: [][]byte{
		{0x06, 0x01, 0x00, 0x00, 0x06, 0x01},
		{0x06, 0x01, 0x00, 0x00, 0x06, 0x01},
	}}
	c := NewClient(ft, 1, 0)
	_ = c.SetBacklight(true)
	_ = c.SetBacklight(false)
	wantOn := frame(t, 1, 0, CmdSetBacklight, BacklightOn)
	wantOff := frame(t, 1, 0, CmdSetBacklight, BacklightOff)
	if !bytes.Equal(ft.sent[0], wantOn) {
		t.Errorf("on:\n got %X\nwant %X", ft.sent[0], wantOn)
	}
	if !bytes.Equal(ft.sent[1], wantOff) {
		t.Errorf("off:\n got %X\nwant %X", ft.sent[1], wantOff)
	}
}

func TestClient_Restart_BadTarget(t *testing.T) {
	ft := &fakeTransport{}
	c := NewClient(ft, 1, 0)
	if err := c.Restart(0x42); err == nil {
		t.Fatal("Restart(0x42) should reject invalid target")
	}
	if len(ft.sent) != 0 {
		t.Fatalf("expected no frame sent on validation failure; got %d", len(ft.sent))
	}
}

func TestClient_Restart_Encoding(t *testing.T) {
	// Spec example: 06 01 00 57 00 50
	ft := &fakeTransport{replies: [][]byte{{0x06, 0x01, 0x00, 0x00, 0x06, 0x01}}}
	c := NewClient(ft, 1, 0)
	if err := c.Restart(RestartAndroid); err != nil {
		t.Fatalf("Restart: %v", err)
	}
	want := []byte{0x06, 0x01, 0x00, 0x57, 0x00, 0x50}
	if !bytes.Equal(ft.sent[0], want) {
		t.Fatalf("sent frame:\n got %X\nwant %X", ft.sent[0], want)
	}
}

func TestClient_AvailableSources(t *testing.T) {
	// Spec example reply (raw bytes from V2.05 §5.5):
	//   11 01 00 AB 0B 0D 06 0F 19 05 0A 10 16 17 11 18 BA
	wire := []byte{0x11, 0x01, 0x00, 0xAB, 0x0B,
		0x0D, 0x06, 0x0F, 0x19, 0x05, 0x0A, 0x10, 0x16, 0x17, 0x11, 0x18, 0xBA}
	ft := &fakeTransport{replies: [][]byte{wire}}
	c := NewClient(ft, 1, 0)
	got, err := c.AvailableSources()
	if err != nil {
		t.Fatalf("AvailableSources: %v", err)
	}
	want := []byte{0x0D, 0x06, 0x0F, 0x19, 0x05, 0x0A, 0x10, 0x16, 0x17, 0x11, 0x18}
	if !bytes.Equal(got, want) {
		t.Fatalf("AvailableSources:\n got %X\nwant %X", got, want)
	}
}

func TestClient_SetVolumePercent_Validation(t *testing.T) {
	ft := &fakeTransport{}
	c := NewClient(ft, 1, 0)
	if err := c.SetVolumePercent(101, 50); err == nil {
		t.Error("speaker=101 should fail")
	}
	if err := c.SetVolumePercent(50, 200); err == nil {
		t.Error("audio_out=200 should fail")
	}
	if len(ft.sent) != 0 {
		t.Fatalf("frames leaked through validation: %d", len(ft.sent))
	}
}

func TestClient_SetVolumePercent_Unchanged(t *testing.T) {
	ft := &fakeTransport{replies: [][]byte{{0x06, 0x01, 0x00, 0x00, 0x06, 0x01}}}
	c := NewClient(ft, 1, 0)
	if err := c.SetVolumePercent(30, VolumeUnchanged); err != nil {
		t.Fatalf("VolumeUnchanged should be accepted: %v", err)
	}
	want := frame(t, 1, 0, CmdSetVolume, 30, VolumeUnchanged)
	if !bytes.Equal(ft.sent[0], want) {
		t.Fatalf("sent frame:\n got %X\nwant %X", ft.sent[0], want)
	}
}

func TestClient_SetSource_Data2IsReservedSentinel(t *testing.T) {
	// V2.05 examples all use Data[2] = 0x09 (reserved playlist sentinel).
	ft := &fakeTransport{replies: [][]byte{{0x06, 0x01, 0x00, 0x00, 0x06, 0x01}}}
	c := NewClient(ft, 1, 0)
	if err := c.SetSource(SrcHDMI, true); err != nil {
		t.Fatal(err)
	}
	want := []byte{0x09, 0x01, 0x00, 0xAC, 0x0D, 0x09, 0x01, 0x00, 0xA1}
	if !bytes.Equal(ft.sent[0], want) {
		t.Fatalf("SetSource HDMI frame:\n got %X\nwant %X", ft.sent[0], want)
	}
}

func TestClient_DisplayOrientation_Roundtrip(t *testing.T) {
	// Spec set example: 0C 01 00 17 00 00 01 00 00 00 00 1B (portrait OSD only)
	ft := &fakeTransport{replies: [][]byte{
		frame(t, 1, 0, CmdGetDisplayOrientation, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00),
		{0x06, 0x01, 0x00, 0x00, 0x06, 0x01},
	}}
	c := NewClient(ft, 1, 0)
	o, err := c.DisplayOrientation()
	if err != nil {
		t.Fatalf("DisplayOrientation: %v", err)
	}
	if o.OSDRotation() != OSDRotationPortrait || o.Window1() != 0x01 {
		t.Errorf("decoded wrong: %+v", o)
	}
	if err := c.SetDisplayOrientation(NewDisplayOrientation(0, 0, 0x01, 0, 0, 0, 0)); err != nil {
		t.Fatal(err)
	}
	want := []byte{0x0C, 0x01, 0x00, 0x17, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x1B}
	if !bytes.Equal(ft.sent[1], want) {
		t.Fatalf("Set orientation frame:\n got %X\nwant %X", ft.sent[1], want)
	}
}

func TestClient_Close(t *testing.T) {
	ft := &fakeTransport{}
	c := NewClient(ft, 1, 0)
	_ = c.Close()
	if !ft.closed {
		t.Fatal("Close() did not propagate")
	}
}
