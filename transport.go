package sicp

import (
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

// Transport sends and receives raw SICP byte frames. Implementations are not
// required to be safe for concurrent use; a Client serialises access for them.
type Transport interface {
	// Send transmits one fully-formed SICP frame (already containing MsgSize
	// and Checksum).
	Send(frame []byte) error

	// Receive reads exactly one SICP frame, blocking until the configured
	// deadline elapses. It must use the MsgSize byte to determine framing.
	Receive() ([]byte, error)

	// Close releases the underlying connection.
	Close() error
}

// DefaultTCPPort is the SICP-over-Ethernet TCP port used by Philips
// professional displays.
const DefaultTCPPort = 5000

// DialTCP opens a TCP connection to the display and returns it as a Transport.
// addr may be "host" (port defaults to 5000) or "host:port". A zero timeout
// disables I/O deadlines.
func DialTCP(addr string, timeout time.Duration) (Transport, error) {
	if _, _, err := net.SplitHostPort(addr); err != nil {
		addr = net.JoinHostPort(addr, fmt.Sprintf("%d", DefaultTCPPort))
	}
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("sicp: dial %s: %w", addr, err)
	}
	return &tcpTransport{conn: conn, timeout: timeout}, nil
}

// NewTransport wraps an existing io.ReadWriteCloser (e.g. a serial port from
// go.bug.st/serial or tarm/serial) as a SICP transport. The returned transport
// reads frames using the leading length byte and applies no timeout.
func NewTransport(rwc io.ReadWriteCloser) Transport {
	return &rwcTransport{rwc: rwc}
}

type tcpTransport struct {
	conn    net.Conn
	timeout time.Duration
}

func (t *tcpTransport) Send(frame []byte) error {
	if t.timeout > 0 {
		_ = t.conn.SetWriteDeadline(time.Now().Add(t.timeout))
	}
	_, err := t.conn.Write(frame)
	return err
}

func (t *tcpTransport) Receive() ([]byte, error) {
	if t.timeout > 0 {
		_ = t.conn.SetReadDeadline(time.Now().Add(t.timeout))
	}
	return readFrame(t.conn)
}

func (t *tcpTransport) Close() error { return t.conn.Close() }

type rwcTransport struct {
	rwc io.ReadWriteCloser
}

func (t *rwcTransport) Send(frame []byte) error {
	_, err := t.rwc.Write(frame)
	return err
}

func (t *rwcTransport) Receive() ([]byte, error) { return readFrame(t.rwc) }

func (t *rwcTransport) Close() error { return t.rwc.Close() }

// readFrame consumes one SICP frame from r: it reads the MsgSize byte, then
// reads the remaining size-1 bytes.
func readFrame(r io.Reader) ([]byte, error) {
	hdr := make([]byte, 1)
	if _, err := io.ReadFull(r, hdr); err != nil {
		return nil, err
	}
	size := int(hdr[0])
	if size < MinFrameSize || size > MaxFrameSize {
		return nil, fmt.Errorf("sicp: invalid MsgSize byte 0x%02X", hdr[0])
	}
	buf := make([]byte, size)
	buf[0] = hdr[0]
	if _, err := io.ReadFull(r, buf[1:]); err != nil {
		if errors.Is(err, io.EOF) {
			err = io.ErrUnexpectedEOF
		}
		return nil, err
	}
	return buf, nil
}
