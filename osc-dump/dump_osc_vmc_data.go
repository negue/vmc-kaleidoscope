package main

import (
	"bufio"
	"compress/gzip"
	"encoding/binary"
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

	out, err := os.Create("dump.bin.gz")
	if err != nil {
		log.Fatal("failed creating output file", err)
	}

	bw := bufio.NewWriter(out)
	gw := gzip.NewWriter(bw)
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

		if _, err := gw.Write(sizeBuf); err != nil {
			log.Fatal("failed writing package size", err)
		}
		if _, err := gw.Write(dataBuf[:n]); err != nil {
			log.Fatal("failed writing package data", err)
		}
	}

	if err := gw.Close(); err != nil {
		log.Fatal("failed closing GZIP writer", err)
	}
	if err := bw.Flush(); err != nil {
		log.Fatal("failed flushing output buffer", err)
	}
	if err := out.Close(); err != nil {
		log.Fatal("failed closing output file", err)
	}

}
