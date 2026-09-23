# sicp

[![Go Reference](https://pkg.go.dev/badge/github.com/go-sicp/sicp.svg)](https://pkg.go.dev/github.com/go-sicp/sicp)

A Go library for the **Philips SICP** (Serial Interface Communication
Protocol), used by Philips professional and digital-signage displays. Based on
*The SICP Commands Document V2.05* (released 2020-11-18). The wire format is
unchanged since V1.88; later revisions add new commands and tighten valid
value ranges (notably volume = 0..100).

The CLI lives in a separate module: [`go-sicp/sicpctl`](https://github.com/go-sicp/sicpctl).

## Install

```sh
go get github.com/go-sicp/sicp
```

## Frame format

```
MsgSize | Control(MonitorID) | Group | Data[0..N] | Checksum (XOR of preceding bytes)
```

* `MsgSize` = total length, 5..40 bytes.
* `MonitorID` 1..255 unicast, 0 = broadcast (no reply).
* `Group` 0 = address by MonitorID, 1..254 = address by group.
* RS-232 default: 9600 8N1. Ethernet default: TCP port **5000**.

## API style

The public surface is a small set of interfaces (`Client`, `Transport`) plus
opaque value types (`Frame`, `VideoParams`) whose fields are unexported.
Callers build values with `NewXxx` constructors and read them through accessor
methods, so the wire format and internal state stay encapsulated.

## Example (TCP)

```go
import "github.com/go-sicp/sicp"

t, _ := sicp.DialTCP("192.168.1.50", 3*time.Second) // port defaults to 5000
defer t.Close()
c := sicp.NewClient(t, /*monitorID*/ 1, /*group*/ 0)

_ = c.PowerOn()
_ = c.SetSource(sicp.SrcHDMI, true)
_ = c.SetVolumePercent(30, sicp.VolumeUnchanged) // 30% speaker, leave audio-out alone
_ = c.SetMute(false)
_ = c.SetBacklight(true)

on, _ := c.Power()
spk, ao, _ := c.VolumePercent()
fmt.Println(on, spk, ao)

inputs, _ := c.AvailableSources() // []byte of source codes the model supports
fmt.Println(inputs)

p, _ := c.Video()
fmt.Println(p.Brightness(), p.Contrast())
_ = c.SetVideo(sicp.NewVideoParams(60, 50, 70, 5, 50, 50, 0x03))

_ = c.Restart(sicp.RestartAndroid) // V2.02+
```

### Capabilities

| Method | SICP opcode | Notes |
|---|---|---|
| `PowerOn`, `PowerOff`, `Power` | `0x18`, `0x19` | |
| `SetSource`, `Source`, `AvailableSources` | `0xAC`, `0xAD`, `0xAB` | `AvailableSources` requires V2.05+ |
| `VolumePercent`, `SetVolumePercent` | `0x44`, `0x45` | 0..100 or `VolumeUnchanged` |
| `Mute`, `SetMute` | `0x46`, `0x47` | V2.00+ |
| `Video`, `SetVideo` | `0x32`, `0x33` | 7-byte payload |
| `Backlight`, `SetBacklight` | `0x71`, `0x72` | V2.02+ |
| `DisplayOrientation`, `SetDisplayOrientation` | `0x16`, `0x17` | V1.90+ |
| `Restart` | `0x57` | V2.02+ |
| `GetModelInfo`, `GetPlatform` | `0xA1`, `0xA2` | |
| `Do(...byte)` | any | low-level escape hatch |

## Example (RS-232)

The library does not depend on a serial-port package; bring your own. Anything
implementing `io.ReadWriteCloser` works:

```go
import (
    "go.bug.st/serial"
    "github.com/go-sicp/sicp"
)

port, _ := serial.Open("/dev/ttyUSB0", &serial.Mode{BaudRate: 9600})
c := sicp.NewClient(sicp.NewTransport(port), 1, 0)
_ = c.PowerOn()
```

## Testing

```sh
go test ./...
```

The test suite encodes / decodes exact byte vectors taken from the *SICP
Commands Document* (V1.88 frame examples + V2.05 examples for the new
backlight, restart and available-sources commands).
