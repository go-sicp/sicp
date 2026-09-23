package sicp

// Command codes (Data[0]) defined by SICP V2.05. The "Get" form returns a
// "Report" frame whose Data[0] echoes the Get code.
const (
	// §2.4 Communication control.
	CmdReportComm byte = 0x00 // Generic ACK/NACK/NAV report

	ReplyACK  byte = 0x06 // Acknowledge
	ReplyNACK byte = 0x15 // Not acknowledged
	ReplyNAV  byte = 0x18 // Not available / unrecognised

	// §3 Platform / model / version.
	// 0xA1 Data[1]: 0x00 model, 0x01 fw, 0x02 build date, 0x03 Android FW build (V2.02+).
	CmdGetModelInfo byte = 0xA1
	// 0xA2 Data[1]: 0x00 SICP version, 0x01 platform label, 0x02 platform version.
	CmdGetPlatform byte = 0xA2

	// §4.1 Power state.
	CmdGetPower byte = 0x19
	CmdSetPower byte = 0x18

	// §4.2 IR / keypad locks.
	CmdGetIRLock     byte = 0x1D
	CmdSetIRLock     byte = 0x1C
	CmdGetKeypadLock byte = 0x1B
	CmdSetKeypadLock byte = 0x1A

	// §4.3 Power state at cold start.
	CmdGetColdStart byte = 0xA4
	CmdSetColdStart byte = 0xA3

	// §4.4 / §5.5 Input source.
	CmdSetSource           byte = 0xAC
	CmdGetSource           byte = 0xAD
	CmdGetAvailableSources byte = 0xAB // V2.05: enumerate all source bytes the model supports

	// §4.5 Auto signal detect / failover.
	CmdGetAutoSignal byte = 0xAF
	CmdSetAutoSignal byte = 0xAE

	// §4.6 / §4.7 V2.02+: monitor restart, backlight on/off.
	CmdMonitorRestart byte = 0x57
	CmdGetBacklight   byte = 0x71
	CmdSetBacklight   byte = 0x72

	// §5.1 Video parameters (brightness, color, contrast, sharpness, tint, black, gamma).
	CmdGetVideo byte = 0x33
	CmdSetVideo byte = 0x32

	// §6.1 Audio.
	CmdGetVolume     byte = 0x45
	CmdSetVolume     byte = 0x44
	CmdStepVolume    byte = 0x41
	CmdGetMute       byte = 0x46 // V2.00+
	CmdSetMute       byte = 0x47 // V2.00+
	CmdGetAudioParam byte = 0x43
	CmdSetAudioParam byte = 0x42

	// §7.10 / §7.11 Display orientation (V1.90+).
	CmdGetDisplayOrientation byte = 0x16
	CmdSetDisplayOrientation byte = 0x17
)

// PowerState values for CmdGetPower / CmdSetPower (§4.1).
const (
	PowerOff byte = 0x01
	PowerOn  byte = 0x02
)

// ColdStart values for CmdGetColdStart / CmdSetColdStart (§4.3).
const (
	ColdStartPowerOff   byte = 0x00
	ColdStartForcedOn   byte = 0x01
	ColdStartLastStatus byte = 0x02
)

// VolumeUnchanged is the sentinel byte that, when passed as the speaker or
// audio-out value of CmdSetVolume, leaves that channel untouched. The display
// ignores any value strictly greater than 100 (0x64); 0xFF is the canonical
// "no change" marker.
const VolumeUnchanged byte = 0xFF

// Backlight values (§4.7). Note the display's convention: 0x00 = ON, 0x01 = OFF.
const (
	BacklightOn  byte = 0x00
	BacklightOff byte = 0x01
)

// Mute values (§6.1.8).
const (
	MuteOff byte = 0x00
	MuteOn  byte = 0x01
)

// MonitorRestart targets for CmdMonitorRestart (§4.6).
const (
	RestartAndroid byte = 0x00
	RestartScalar  byte = 0x01
)

// Source codes for CmdSetSource / CmdGetSource (§4.4) — including those added
// in V1.89..V2.05 (Media Player, PDF Player, Custom, HDMI4, VGA2, VGA3, IWB,
// CMND&Play Web).
const (
	SrcVideo           byte = 0x01
	SrcSVideo          byte = 0x02
	SrcComponent       byte = 0x03
	SrcVGA             byte = 0x05
	SrcHDMI2           byte = 0x06
	SrcDisplayPort2    byte = 0x07
	SrcUSB2            byte = 0x08
	SrcCardDVID        byte = 0x09
	SrcDisplayPort1    byte = 0x0A
	SrcCardOPS         byte = 0x0B
	SrcUSB1            byte = 0x0C
	SrcHDMI            byte = 0x0D
	SrcDVID            byte = 0x0E
	SrcHDMI3           byte = 0x0F
	SrcBrowser         byte = 0x10
	SrcSmartCMS        byte = 0x11
	SrcDMS             byte = 0x12
	SrcInternalStorage byte = 0x13
	SrcMediaPlayer     byte = 0x16 // V1.89+
	SrcPDFPlayer       byte = 0x17 // V1.89+
	SrcCustom          byte = 0x18 // V1.89+
	SrcHDMI4           byte = 0x19 // V1.99+
	SrcVGA2            byte = 0x1A // V1.99+
	SrcVGA3            byte = 0x1B // V1.99+
	SrcIWB             byte = 0x1C // V1.99+
	SrcCMNDPlayWeb     byte = 0x1D // V2.04+
)

// SourceName maps an input-source byte to its human label.
func SourceName(b byte) string {
	switch b {
	case SrcVideo:
		return "VIDEO"
	case SrcSVideo:
		return "S-VIDEO"
	case SrcComponent:
		return "COMPONENT"
	case SrcVGA:
		return "VGA"
	case SrcHDMI:
		return "HDMI"
	case SrcHDMI2:
		return "HDMI2"
	case SrcHDMI3:
		return "HDMI3"
	case SrcHDMI4:
		return "HDMI4"
	case SrcDVID:
		return "DVI-D"
	case SrcCardDVID:
		return "CardDVI-D"
	case SrcDisplayPort1:
		return "DisplayPort1"
	case SrcDisplayPort2:
		return "DisplayPort2"
	case SrcVGA2:
		return "VGA2"
	case SrcVGA3:
		return "VGA3"
	case SrcUSB1:
		return "USB1"
	case SrcUSB2:
		return "USB2"
	case SrcCardOPS:
		return "CardOPS"
	case SrcBrowser:
		return "Browser"
	case SrcSmartCMS:
		return "SmartCMS"
	case SrcDMS:
		return "DMS"
	case SrcInternalStorage:
		return "InternalStorage"
	case SrcMediaPlayer:
		return "MediaPlayer"
	case SrcPDFPlayer:
		return "PDFPlayer"
	case SrcCustom:
		return "Custom"
	case SrcIWB:
		return "IWB"
	case SrcCMNDPlayWeb:
		return "CMNDPlayWeb"
	default:
		return ""
	}
}

// allSources lists every byte SourceName recognises. Kept in one place so
// SourceFromName stays in sync with SourceName automatically.
var allSources = []byte{
	SrcVideo, SrcSVideo, SrcComponent, SrcVGA,
	SrcHDMI, SrcHDMI2, SrcHDMI3, SrcHDMI4,
	SrcDVID, SrcCardDVID,
	SrcDisplayPort1, SrcDisplayPort2,
	SrcVGA2, SrcVGA3,
	SrcUSB1, SrcUSB2, SrcCardOPS,
	SrcBrowser, SrcSmartCMS, SrcDMS, SrcInternalStorage,
	SrcMediaPlayer, SrcPDFPlayer, SrcCustom, SrcIWB, SrcCMNDPlayWeb,
}

// SourceFromName parses a case-insensitive label produced by SourceName.
// Returns 0 and false for unknown labels.
func SourceFromName(name string) (byte, bool) {
	for _, b := range allSources {
		if eqFold(SourceName(b), name) {
			return b, true
		}
	}
	return 0, false
}

func eqFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
