package main

import (
	"bufio"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Open UDP listener connection.
	conn, err := net.ListenPacket("udp", ":39543")
	if err != nil {
		log.Fatal("failed listening on UDP", err)
	}

	// Create output file, which is GZIP-compressed and swapped automatically, once it reaches
	// the given size limit (currently 8 MB).
	out, err := newSwapWriter(8 * 1024 * 1024)
	if err != nil {
		log.Fatal("failed creating output file writer", err)
	}

	// Install a shutdown handler so we can cleanly shut down and close our output file properly.
	// Also, allocate some buffers for reading/saving UDP data.
	shutdown := installShutdownHandler()
	sizeBuf := make([]byte, 2)
	dataBuf := make([]byte, 65535)

loop:
	// Main loop, reading data endlessly until we receive a shutdown signal.
	for {
		select {
		// Got a shutdown signal, lets break out of the read loop.
		case <-shutdown:
			break loop
		// Try to read a new UDP packet, stopping the loop if we failed.
		default:
			if err := logData(conn, out, sizeBuf, dataBuf); err != nil {
				log.Println("failed logging data", err)
				break loop
			}
		}
	}

	// Close the output file first, so it doesn't turn out as a corrupt GZIP file.
	if err := out.Close(); err != nil {
		log.Fatal("failed closing output file writer", err)
	}

	// Then close the UDP connection.
	if err := conn.Close(); err != nil {
		log.Fatal("failed closing UDP connection", err)
	}
}

func installShutdownHandler() <-chan struct{} {
	c := make(chan os.Signal, 1)
	n := make(chan struct{}, 1)

	// Install OS signal handlers
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Spawn a goroutine that will wait for the OS signal and
	// then notify back on the channel n.
	go func() {
		<-c
		log.Println("shutting down, please wait...")
		n <- struct{}{}
		close(n)
	}()

	return n
}

func logData(conn net.PacketConn, out io.Writer, dataBuf, sizeBuf []byte) error {
	// Set a deadline, so we don't block forever if no data comes in at all.
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return fmt.Errorf("failed setting read deadline: %w", err)
	}

	n, _, err := conn.ReadFrom(dataBuf)
	if errors.Is(err, os.ErrDeadlineExceeded) {
		log.Println("no data within the last 5 seconds, are we connected?")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed reading from UDP: %w", err)
	}

	binary.BigEndian.PutUint16(sizeBuf, uint16(n))

	if _, err := out.Write(sizeBuf); err != nil {
		return fmt.Errorf("failed writing package size: %w", err)
	}
	if _, err := out.Write(dataBuf[:n]); err != nil {
		return fmt.Errorf("failed writing package data: %w", err)
	}

	return nil
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
