package minecraft

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"strings"
	"testing"
	"time"
)

// blackholeConn accepts every write and never delivers anything back, like a client that stopped answering
// once the start game sequence was sent to it.
type blackholeConn struct {
	closed chan struct{}
}

func newBlackholeConn() *blackholeConn {
	return &blackholeConn{closed: make(chan struct{})}
}

func (c *blackholeConn) Read([]byte) (int, error) {
	<-c.closed
	return 0, io.EOF
}

func (c *blackholeConn) Write(b []byte) (int, error) { return len(b), nil }

func (c *blackholeConn) Close() error {
	select {
	case <-c.closed:
	default:
		close(c.closed)
	}
	return nil
}

func (c *blackholeConn) LocalAddr() net.Addr  { return &net.UDPAddr{IP: net.IPv4zero} }
func (c *blackholeConn) RemoteAddr() net.Addr { return &net.UDPAddr{IP: net.IPv4(1, 2, 3, 4)} }

func (c *blackholeConn) SetDeadline(time.Time) error      { return nil }
func (c *blackholeConn) SetReadDeadline(time.Time) error  { return nil }
func (c *blackholeConn) SetWriteDeadline(time.Time) error { return nil }

func newTestConn(t *testing.T, netConn net.Conn) *Conn {
	t.Helper()
	conn := newConn(netConn, nil, slog.New(slog.DiscardHandler), DefaultProtocol, time.Second, false)
	t.Cleanup(func() { _ = netConn.Close() })
	return conn
}

// TestStartGameTimeoutReportsClientActivity asserts the spawn timeout says whether the client answered at
// all, which is what separates a client that went silent from one stuck later in the sequence.
func TestStartGameTimeoutReportsClientActivity(t *testing.T) {
	conn := newTestConn(t, newBlackholeConn())

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := conn.StartGameContext(ctx, GameData{WorldName: "test"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected a deadline error, got %v", err)
	}
	if !strings.Contains(err.Error(), "0 packets from client since start game") {
		t.Fatalf("expected the error to report the silent client, got %v", err)
	}
}
