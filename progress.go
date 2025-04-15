// this file provides common functionnalities that aims to
// send through a channel the progression of an upload and download
// Note: We use it with a tee reader, if progressWriter blocks, the other
// side of the tee reader (eg io.Copy on tcp conn) will also block
package shair

type ProgressWriter struct {
	ProgressCh chan<- int
}

func NewProgressWriter(progressCh chan<- int) *ProgressWriter {
	return &ProgressWriter{
		ProgressCh: progressCh,
	}
}

func (w *ProgressWriter) Write(p []byte) (int, error) {
	w.ProgressCh <- len(p)
	return len(p), nil
}
