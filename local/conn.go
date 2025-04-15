package local

import (
	"context"
	"net"
)

// this struct provides an implementation of a context-aware net.Conn
// it wraps a net.Conn and is able to cancel any io.Copy / Write process
// when its context is cancelled
type contextConn struct {
	ctx  context.Context
	conn net.Conn
}

func newContextWriter(ctx context.Context, c net.Conn) contextConn {
	return contextConn{
		ctx:  ctx,
		conn: c,
	}
}

func (w contextConn) Write(p []byte) (int, error) {
	select {
	case <-w.ctx.Done():
		return 0, w.ctx.Err()

	default:
		return w.conn.Write(p)
	}
}
