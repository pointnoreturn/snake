package libradios

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"time"
)

// basic IO primitive for serial OR tcp stream
type NetStream struct {
	Transport
	netPort net.Conn
}

func (s *NetStream) CanRead() bool { return true }

func (s *NetStream) CanWrite() bool { return true }

func (s *NetStream) Close() {
	if s.netPort != nil {
		s.netPort.Close()
	}
}

func NewNetStream(
	ctx context.Context,
	addr string,
	defaultPort string,
) (*NetStream, error) {

	tcpAddr, ok := ParseTCPAddress(addr, defaultPort)
	if !ok {
		s := fmt.Sprintf("Failed to parse TCP address for NetStream connect: '%s'", addr)
		return nil, errors.New(s)
	}

	fmt.Println("[BaseStream] Connecting as TCP")

	d := net.Dialer{}

	conn, err := d.DialContext(
		ctx,
		"tcp",
		tcpAddr,
	)
	if err != nil {
		return nil, err
	}

	return &NetStream{netPort: conn}, nil
}

func (s *NetStream) Write(
	ctx context.Context,
	p []byte,
) error {

	written := 0

	for written < len(p) {

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := s.netPort.SetWriteDeadline(
			time.Now().Add(200 * time.Millisecond),
		)
		if err != nil {
			return err
		}

		n, err := s.netPort.Write(p[written:])

		if err != nil {

			if errors.Is(err, os.ErrDeadlineExceeded) {
				continue
			}

			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}

			return err
		}

		written += n
	}

	return nil
}

func (s *NetStream) Read(ctx context.Context, p []byte) (int, error) {

	for {

		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}

		// short polling deadline
		s.netPort.SetReadDeadline(time.Now().Add(200 * time.Millisecond))

		n, err := s.netPort.Read(p)

		if err != nil {

			// timeout is expected, continue polling
			if errors.Is(err, os.ErrDeadlineExceeded) {
				continue
			}

			netErr, ok := err.(net.Error)
			if ok && netErr.Timeout() {
				continue
			}

			return n, err
		}

		return n, nil
	}
}
