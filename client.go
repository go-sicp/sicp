package sicp

import (
	"errors"
	"fmt"
	"sync"
)

// Client is the high-level SICP API. It is bound to a single transport and a
// single target monitor address, and it serialises requests so it is safe to
// share across goroutines. Obtain one through NewClient; the underlying type
// is unexported.
type Client interface {
	// Power state (§4.1).
	PowerOn() error
	PowerOff() error
	Power() (on bool, err error)

	// Input source (§4.4 / §5.5).
	SetSource(src byte, showOnOSD bool) error
	Source() (byte, error)
	AvailableSources() ([]byte, error)

	// Volume / mute (§6.1).
	VolumePercent() (speaker, audioOut byte, err error)
	SetVolumePercent(speaker, audioOut byte) error
	Mute() (bool, error)
	SetMute(on bool) error

	// Video parameters (§5.1).
	Video() (VideoParams, error)
	SetVideo(p VideoParams) error

	// Backlight on/off (§4.7, V2.02+).
	Backlight() (on bool, err error)
	SetBacklight(on bool) error

	// Display orientation (§7.10/§7.11, V1.90+).
	DisplayOrientation() (DisplayOrientation, error)
	SetDisplayOrientation(o DisplayOrientation) error

	// Restart the monitor (§4.6, V2.02+). target is RestartAndroid or
	// RestartScalar. The display ACKs and reboots; subsequent calls will fail
	// until it comes back.
	Restart(target byte) error

	// Identity / version strings (§3).
	GetModelInfo(kind byte) (string, error)
	GetPlatform(kind byte) (string, error)

	// Do is the low-level escape hatch. data must start with the command byte
	// (Data[0]). It returns the data payload of the report frame, or nil when
	// the monitor ID is Broadcast.
	Do(data ...byte) ([]byte, error)

	// Close releases the underlying transport.
	Close() error
}

// NewClient builds a client targeting monitorID (1..255 unicast, 0 broadcast)
// on the given group (0 = address by monitor ID).
func NewClient(t Transport, monitorID, group byte) Client {
	return &client{t: t, monitorID: monitorID, group: group}
}

type client struct {
	t         Transport
	monitorID byte
	group     byte
	mu        sync.Mutex
}

func (c *client) Close() error { return c.t.Close() }

func (c *client) Do(data ...byte) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	frame, err := NewFrame(c.monitorID, c.group, data)
	if err != nil {
		return nil, err
	}
	wire, err := frame.Encode()
	if err != nil {
		return nil, err
	}
	if err := c.t.Send(wire); err != nil {
		return nil, fmt.Errorf("sicp: send: %w", err)
	}
	if c.monitorID == Broadcast {
		return nil, nil
	}
	raw, err := c.t.Receive()
	if err != nil {
		return nil, fmt.Errorf("sicp: receive: %w", err)
	}
	rep, err := Decode(raw)
	if err != nil {
		return nil, err
	}
	d := rep.Data()
	if err := checkReportError(d); err != nil {
		return nil, err
	}
	return d, nil
}

// checkReportError translates a generic communication-control report
// (Data[0]=0x00) into a Go error. For non-generic reports it returns nil.
func checkReportError(data []byte) error {
	if len(data) < 2 || data[0] != CmdReportComm {
		return nil
	}
	switch data[1] {
	case ReplyACK:
		return nil
	case ReplyNACK:
		return errors.New("sicp: NACK (command rejected)")
	case ReplyNAV:
		return errors.New("sicp: NAV (command not available / wrong checksum / not executable)")
	default:
		return fmt.Errorf("sicp: unknown report code 0x%02X", data[1])
	}
}

// ---------------------------------------------------------------------------
// High-level helpers.
// ---------------------------------------------------------------------------

func (c *client) PowerOn() error  { _, err := c.Do(CmdSetPower, PowerOn); return err }
func (c *client) PowerOff() error { _, err := c.Do(CmdSetPower, PowerOff); return err }

func (c *client) Power() (bool, error) {
	data, err := c.Do(CmdGetPower)
	if err != nil {
		return false, err
	}
	if len(data) < 2 {
		return false, fmt.Errorf("sicp: short power report (%d bytes)", len(data))
	}
	return data[1] == PowerOn, nil
}

func (c *client) SetSource(src byte, showOnOSD bool) error {
	osd := byte(0x00)
	if showOnOSD {
		osd = 0x01
	}
	// Data[2] is "playlist / URL number" since V1.89; 0x09..0x18 are reserved
	// values meaning "no playlist", which every V2.05 example uses verbatim.
	// Data[3] = OSD style, Data[4] = mute style.
	_, err := c.Do(CmdSetSource, src, 0x09, osd, 0x00)
	return err
}

func (c *client) AvailableSources() ([]byte, error) {
	data, err := c.Do(CmdGetAvailableSources)
	if err != nil {
		return nil, err
	}
	if len(data) < 2 {
		return nil, fmt.Errorf("sicp: short available-sources report (%d bytes)", len(data))
	}
	n := int(data[1])
	if len(data) < 2+n {
		return nil, fmt.Errorf("sicp: truncated available-sources report: header says %d, payload has %d", n, len(data)-2)
	}
	out := make([]byte, n)
	copy(out, data[2:2+n])
	return out, nil
}

func (c *client) Source() (byte, error) {
	data, err := c.Do(CmdGetSource)
	if err != nil {
		return 0, err
	}
	if len(data) < 2 {
		return 0, fmt.Errorf("sicp: short source report (%d bytes)", len(data))
	}
	return data[1], nil
}

func (c *client) VolumePercent() (speaker, audioOut byte, err error) {
	data, e := c.Do(CmdGetVolume)
	if e != nil {
		return 0, 0, e
	}
	if len(data) < 3 {
		return 0, 0, fmt.Errorf("sicp: short volume report (%d bytes)", len(data))
	}
	return data[1], data[2], nil
}

// SetVolumePercent sets the speaker and audio-out volumes (§6.1, V1.98+).
// Each value must be 0..100 or VolumeUnchanged (0xFF) to leave that channel
// alone. Anything else returns an error without sending the frame, since the
// display silently ignores values > 100.
func (c *client) SetVolumePercent(speaker, audioOut byte) error {
	if err := validateVolume(speaker, "speaker"); err != nil {
		return err
	}
	if err := validateVolume(audioOut, "audio_out"); err != nil {
		return err
	}
	_, err := c.Do(CmdSetVolume, speaker, audioOut)
	return err
}

func validateVolume(v byte, name string) error {
	if v > 100 && v != VolumeUnchanged {
		return fmt.Errorf("sicp: %s volume %d out of range (want 0..100 or VolumeUnchanged)", name, v)
	}
	return nil
}

func (c *client) Mute() (bool, error) {
	data, err := c.Do(CmdGetMute)
	if err != nil {
		return false, err
	}
	if len(data) < 2 {
		return false, fmt.Errorf("sicp: short mute report (%d bytes)", len(data))
	}
	return data[1] == MuteOn, nil
}

func (c *client) SetMute(on bool) error {
	v := MuteOff
	if on {
		v = MuteOn
	}
	_, err := c.Do(CmdSetMute, v)
	return err
}

func (c *client) Backlight() (bool, error) {
	data, err := c.Do(CmdGetBacklight)
	if err != nil {
		return false, err
	}
	if len(data) < 2 {
		return false, fmt.Errorf("sicp: short backlight report (%d bytes)", len(data))
	}
	return data[1] == BacklightOn, nil
}

func (c *client) SetBacklight(on bool) error {
	v := BacklightOff
	if on {
		v = BacklightOn
	}
	_, err := c.Do(CmdSetBacklight, v)
	return err
}

func (c *client) Restart(target byte) error {
	if target != RestartAndroid && target != RestartScalar {
		return fmt.Errorf("sicp: invalid restart target 0x%02X", target)
	}
	_, err := c.Do(CmdMonitorRestart, target)
	return err
}

func (c *client) Video() (VideoParams, error) {
	data, err := c.Do(CmdGetVideo)
	if err != nil {
		return VideoParams{}, err
	}
	if len(data) < 8 {
		return VideoParams{}, fmt.Errorf("sicp: short video report (%d bytes)", len(data))
	}
	return NewVideoParams(data[1], data[2], data[3], data[4], data[5], data[6], data[7]), nil
}

func (c *client) SetVideo(p VideoParams) error {
	_, err := c.Do(CmdSetVideo,
		p.Brightness(), p.Color(), p.Contrast(), p.Sharpness(),
		p.Tint(), p.BlackLevel(), p.Gamma())
	return err
}

// ModelInfo identifier kinds for GetModelInfo (§3).
const (
	ModelInfoNumber    byte = 0x00 // Model number
	ModelInfoFW        byte = 0x01 // Firmware version
	ModelInfoBuildDate byte = 0x02 // Build date
	ModelInfoAndroidFW byte = 0x03 // Android FW build number (V2.02+)
)

func (c *client) GetModelInfo(kind byte) (string, error) {
	data, err := c.Do(CmdGetModelInfo, kind)
	if err != nil {
		return "", err
	}
	if len(data) < 2 || data[0] != CmdGetModelInfo {
		return "", fmt.Errorf("sicp: unexpected model report")
	}
	return string(data[1:]), nil
}

// PlatformInfo identifier kinds for GetPlatform (§3.2).
const (
	PlatformInfoSICPVersion byte = 0x00
	PlatformInfoLabel       byte = 0x01
	PlatformInfoVersion     byte = 0x02
)

func (c *client) GetPlatform(kind byte) (string, error) {
	data, err := c.Do(CmdGetPlatform, kind)
	if err != nil {
		return "", err
	}
	if len(data) < 2 || data[0] != CmdGetPlatform {
		return "", fmt.Errorf("sicp: unexpected platform report")
	}
	return string(data[1:]), nil
}

// VideoParams holds the seven values of the §6.1 video-parameter command. Its
// fields are unexported; build one with NewVideoParams and read them through
// the accessor methods.
type VideoParams struct {
	brightness byte
	color      byte
	contrast   byte
	sharpness  byte
	tint       byte
	blackLevel byte
	gamma      byte
}

// NewVideoParams builds a VideoParams. Most values are 0..100; gamma is
// 0x01 native, 0x02 S, 0x03 2.2, 0x04 2.4, 0x05 D-image. tint is 0..100 on
// most platforms or signed -50..50 (sent as a raw byte) on Phoenix 2.0.
func NewVideoParams(brightness, color, contrast, sharpness, tint, blackLevel, gamma byte) VideoParams {
	return VideoParams{
		brightness: brightness,
		color:      color,
		contrast:   contrast,
		sharpness:  sharpness,
		tint:       tint,
		blackLevel: blackLevel,
		gamma:      gamma,
	}
}

func (p VideoParams) Brightness() byte { return p.brightness }
func (p VideoParams) Color() byte      { return p.color }
func (p VideoParams) Contrast() byte   { return p.contrast }
func (p VideoParams) Sharpness() byte  { return p.sharpness }
func (p VideoParams) Tint() byte       { return p.tint }
func (p VideoParams) BlackLevel() byte { return p.blackLevel }
func (p VideoParams) Gamma() byte      { return p.gamma }

// OSDRotation values for DisplayOrientation.osdRotation (§7.10/§7.11).
const (
	OSDRotationLandscape byte = 0x00
	OSDRotationPortrait  byte = 0x01
)

// ImageRotation values for DisplayOrientation.imageRotation (§7.10/§7.11).
const (
	ImageRotationOff             byte = 0x00
	ImageRotationOn              byte = 0x01 // not supported on CRD50
	ImageRotationClockwise       byte = 0x02 // CRD50 only
	ImageRotationCounterCockwise byte = 0x03 // CRD50 only
)

// DisplayOrientation packs the seven data bytes of the V1.90+ display-
// orientation Get/Set command (§7.11). Fields are unexported; build one with
// NewDisplayOrientation and read them through the accessors.
type DisplayOrientation struct {
	autoRotate    byte // 0x00 off / 0x01 on (Dragon 1.x only)
	osdRotation   byte // OSDRotation*
	imageRotation byte // ImageRotation*
	window1       byte // 0x00 off / 0x01 on  (main)
	window2       byte // sub 1
	window3       byte // sub 2
	window4       byte // sub 3
}

// NewDisplayOrientation builds a DisplayOrientation. Each window flag is
// 0x00 off or 0x01 on.
func NewDisplayOrientation(autoRotate, osdRotation, imageRotation, w1, w2, w3, w4 byte) DisplayOrientation {
	return DisplayOrientation{
		autoRotate: autoRotate, osdRotation: osdRotation, imageRotation: imageRotation,
		window1: w1, window2: w2, window3: w3, window4: w4,
	}
}

func (o DisplayOrientation) AutoRotate() byte    { return o.autoRotate }
func (o DisplayOrientation) OSDRotation() byte   { return o.osdRotation }
func (o DisplayOrientation) ImageRotation() byte { return o.imageRotation }
func (o DisplayOrientation) Window1() byte       { return o.window1 }
func (o DisplayOrientation) Window2() byte       { return o.window2 }
func (o DisplayOrientation) Window3() byte       { return o.window3 }
func (o DisplayOrientation) Window4() byte       { return o.window4 }

func (c *client) DisplayOrientation() (DisplayOrientation, error) {
	data, err := c.Do(CmdGetDisplayOrientation)
	if err != nil {
		return DisplayOrientation{}, err
	}
	if len(data) < 8 {
		return DisplayOrientation{}, fmt.Errorf("sicp: short display-orientation report (%d bytes)", len(data))
	}
	return NewDisplayOrientation(data[1], data[2], data[3], data[4], data[5], data[6], data[7]), nil
}

func (c *client) SetDisplayOrientation(o DisplayOrientation) error {
	_, err := c.Do(CmdSetDisplayOrientation,
		o.autoRotate, o.osdRotation, o.imageRotation,
		o.window1, o.window2, o.window3, o.window4)
	return err
}
