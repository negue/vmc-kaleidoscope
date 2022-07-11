package main

import (
	"bufio"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

func main() {
	conn, err := net.ListenPacket("udp", ":39543")
	if err != nil {
		log.Fatal("failed listening on UDP", err)
	}

	out, err := newSwapWriter(8 * 1024 * 1024)
	if err != nil {
		log.Fatal("failed creating output file writer", err)
	}
	defer out.Close()

	start := time.Now()
	sizeBuf := make([]byte, 2)
	dataBuf := make([]byte, 65535)

	for time.Since(start) < 5*time.Minute {
		if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
			log.Fatal("failed setting read deadline", err)
		}

		n, _, err := conn.ReadFrom(dataBuf)
		if err != nil {
			log.Fatal("failed reading from UDP")
		}

		binary.BigEndian.PutUint16(sizeBuf, uint16(n))

		if _, err := out.Write(sizeBuf); err != nil {
			log.Fatal("failed writing package size", err)
		}
		if _, err := out.Write(dataBuf[:n]); err != nil {
			log.Fatal("failed writing package data", err)
		}
	}

	if err := out.Close(); err != nil {
		log.Fatal("failed closing output file writer", err)
	}
}

type swapWriter struct {
	current *logWriter
	limit   int
	written int
	count   int
}

func newSwapWriter(limit int) (*swapWriter, error) {
	w := &swapWriter{limit: limit}
	if err := w.swap(); err != nil {
		return nil, err
	}

	return w, nil
}

func (w *swapWriter) swap() error {
	if w.current != nil {
		if err := w.current.Close(); err != nil {
			return fmt.Errorf("failed closing current writer: %w", err)
		}
	}

	w.count += 1
	w.written = 0

	fileName := fmt.Sprintf("dump-%05d.bin.gz", w.count)
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}

	w.current = newLogWriter(file)

	return nil
}

var _ io.WriteCloser = (*swapWriter)(nil)

func (w *swapWriter) Write(p []byte) (int, error) {
	if len(p)+w.written >= w.limit {
		if err := w.swap(); err != nil {
			return 0, err
		}
	}

	return w.current.Write(p)
}

func (w *swapWriter) Close() error {
	return w.current.Close()
}

type logWriter struct {
	gzip *gzip.Writer
	buf  *bufio.Writer
	out  *os.File
}

func newLogWriter(out *os.File) *logWriter {
	bw := bufio.NewWriter(out)
	gw := gzip.NewWriter(bw)

	return &logWriter{
		gzip: gw,
		buf:  bw,
		out:  out,
	}
}

var _ io.WriteCloser = (*logWriter)(nil)

func (w *logWriter) Write(p []byte) (int, error) {
	return w.gzip.Write(p)
}

func (w *logWriter) Close() error {
	if err := w.gzip.Close(); err != nil {
		return fmt.Errorf("failed closing GZIP writer: %w", err)
	}
	if err := w.buf.Flush(); err != nil {
		return fmt.Errorf("failed flushing output buffer: %w", err)
	}
	if err := w.out.Close(); err != nil {
		return fmt.Errorf("failed closing output file: %w", err)
	}

	return nil
}
