package local

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/masar3141/shair"
)

type sender struct {
	ctxConn contextConn
	files   []*os.File
}

func newSender(ctx context.Context, conn net.Conn, f []*os.File) sender {
	return sender{
		ctxConn: newContextWriter(ctx, conn),
		files:   f,
	}
}

func (s sender) writeHeaderToConn(hdr *header) (int, error) {
	bhdr := hdr.encode()

	written := 0
	for written < int(hdr.headerSize) {
		n, err := s.ctxConn.Write(bhdr[written:])
		written += n

		if err != nil {
			if errors.Is(err, context.Canceled) {
				return written, fmt.Errorf("context cancelled while sending header: %w", err)

			} else {
				return written, fmt.Errorf("can't send the header %w", err)
			}
		}
	}

	return written, nil
}

// send all files on a tcp connection
func (s sender) sendFiles(uploadProgressCh chan<- int) error {
	defer s.ctxConn.conn.Close()

	for i := 0; i < len(s.files); i++ {
		_, err := s.sendFile(i, uploadProgressCh)
		if err != nil {
			return err
		}
	}

	close(uploadProgressCh)

	return nil
}

func (s sender) sendFile(fileNumber int, uploadProgressCh chan<- int) (int64, error) {
	file := s.files[fileNumber]

	multiWriter := io.MultiWriter(s.ctxConn, shair.NewProgressWriter(uploadProgressCh))

	n, err := io.Copy(multiWriter, s.files[fileNumber])

	if err != nil {
		if errors.Is(err, context.Canceled) {
			return n, fmt.Errorf("context cancelled while sending file %s: %w", file.Name(), err)

		} else if errors.Is(err, io.EOF) {
			return n, fmt.Errorf("connection closed while sending file %s: %w", file.Name(), err)

		} else {
			return n, fmt.Errorf("can't send file %s: %w", file.Name(), err)
		}
	}

	return n, nil
}
