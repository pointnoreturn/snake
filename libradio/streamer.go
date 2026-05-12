package libradio

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"time"

	"github.com/jacobsa/go-serial/serial"
)

type streamer struct {
	serialPort io.ReadWriteCloser
	netPort    net.Conn
	isTCP      bool
}

func (s *streamer) Init(addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		// If SplitHostPort fails, it's likely because no port was provided
		// Treat the whole addr as the host and use your default port
		host = addr
		if port == "" {
			port = "4403" // meshtastic
		}
	}

	// Resolve the address string into a TCP address
	tcpAddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(host, port))
	if err == nil {
		conn, err := net.DialTCP("tcp", nil, tcpAddr)
		if err != nil {
			return err
		}
		s.netPort = conn
		s.isTCP = true
	} else {
		//Configure the serial port
		options := serial.OpenOptions{
			PortName:              addr,
			BaudRate:              115200,
			DataBits:              8,
			StopBits:              1,
			MinimumReadSize:       0,
			InterCharacterTimeout: 100,
			ParityMode:            serial.PARITY_NONE,
		}

		// Open the port.
		port, err := serial.Open(options)
		if err != nil {
			return err
		}

		s.serialPort = port
		s.isTCP = false

		return nil
	}

	return nil
}

func ParseTCPAddress(addr string) (string, bool) {

	const defaultPort = "4403"

	// Case:
	// [ipv6]:port
	// ipv4:port
	host, port, err := net.SplitHostPort(addr)

	if err == nil {

		if ip := net.ParseIP(host); ip != nil {

			if port == "" {
				port = defaultPort
			}

			return net.JoinHostPort(host, port), true
		}
	}

	// Plain IP without port
	if ip := net.ParseIP(addr); ip != nil {

		return net.JoinHostPort(addr, defaultPort), true
	}

	return "", false
}

func (s *streamer) InitContext(
	ctx context.Context,
	addr string,
) error {

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if tcpAddr, ok := ParseTCPAddress(addr); ok {

		d := net.Dialer{}

		conn, err := d.DialContext(
			ctx,
			"tcp",
			tcpAddr,
		)
		if err != nil {
			return err
		}

		s.netPort = conn
		s.isTCP = true

		return nil
	}

	// serial path

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	options := serial.OpenOptions{
		PortName:              addr,
		BaudRate:              115200,
		DataBits:              8,
		StopBits:              1,
		MinimumReadSize:       0,
		InterCharacterTimeout: 100,
		ParityMode:            serial.PARITY_NONE,
	}

	type result struct {
		port io.ReadWriteCloser
		err  error
	}

	ch := make(chan result, 1)

	go func() {
		p, err := serial.Open(options)
		ch <- result{
			port: p,
			err:  err,
		}
	}()

	select {

	case <-ctx.Done():
		return ctx.Err()

	case res := <-ch:

		if res.err != nil {
			return res.err
		}

		s.serialPort = res.port
		s.isTCP = false

		return nil
	}
}

func (s *streamer) Write(p []byte) error {

	if s.isTCP {
		s.netPort.SetReadDeadline(time.Now().Add(1 * time.Second))
		_, err := s.netPort.Write(p)
		if err != nil {
			return err
		}
	} else {
		_, err := s.serialPort.Write(p)
		if err != nil {
			return err
		}

		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

func (s *streamer) WriteContext(
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

		if s.isTCP {

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

		} else {

			// serial libraries vary a lot
			if dl, ok := s.serialPort.(interface {
				SetWriteDeadline(time.Time) error
			}); ok {
				_ = dl.SetWriteDeadline(
					time.Now().Add(200 * time.Millisecond),
				)
			}

			n, err := s.serialPort.Write(p[written:])

			if err != nil {

				if errors.Is(err, os.ErrDeadlineExceeded) {
					continue
				}

				return err
			}

			written += n

			// optional throttling for serial radios
			time.Sleep(5 * time.Millisecond)
		}
	}

	return nil
}

func (s *streamer) Read(p []byte) error {

	if s.isTCP {
		s.netPort.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, err := s.netPort.Read(p)
		if err != nil {
			return err
		}
	} else {
		_, err := s.serialPort.Read(p)
		if err != nil {
			return err
		}
	}

	return nil

}

func (s *streamer) ReadContext(ctx context.Context, p []byte) error {

	for {

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if s.isTCP {

			// short polling deadline
			s.netPort.SetReadDeadline(time.Now().Add(200 * time.Millisecond))

			_, err := s.netPort.Read(p)

			if err != nil {

				// timeout is expected, continue polling
				if errors.Is(err, os.ErrDeadlineExceeded) {
					continue
				}

				netErr, ok := err.(net.Error)
				if ok && netErr.Timeout() {
					continue
				}

				return err
			}

			return nil

		} else {

			// serial ports usually support deadlines too
			if dl, ok := s.serialPort.(interface {
				SetReadDeadline(time.Time) error
			}); ok {
				dl.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
			}

			_, err := s.serialPort.Read(p)

			if err != nil {

				if errors.Is(err, os.ErrDeadlineExceeded) {
					continue
				}

				return err
			}

			return nil
		}
	}
}

func (s *streamer) Close() {
	if s.isTCP {
		s.netPort.Close()
	} else {
		s.serialPort.Close()
	}
}
