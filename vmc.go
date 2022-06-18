package main

import (
	"bufio"
	"bytes"
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/g3n/engine/math32"

	osc "just-do-it/osc"
)

// Go Cons:
// - no string concat template way
// - weird fixed array variable declarion syntax  var myArray[4]int

// maybe?
// - memory in fmt.println when called a bunch of times

func main() {
	rand.Seed(time.Now().UnixNano())

	config, err := ReadConfig()

	if err != nil {
		fmt.Printf("Some error %v", err)
		return
	}

	connClient, err := net.Dial("udp", "127.0.0.1:39544")
	if err != nil {
		fmt.Printf("Some error %v", err)
		return
	}

	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		Port: int(config.ListenTo),
		IP:   net.ParseIP("0.0.0.0"),
	})
	if err != nil {
		panic(err)
	}

	defer conn.Close()
	fmt.Printf("server listening %s\n", conn.LocalAddr().String())

	for {
		rawMessage := make([]byte, 2000)
		rlen, _, err := conn.ReadFromUDP(rawMessage[:])
		if err != nil {
			panic(err)
		}

		clippedRawBytes := rawMessage[0:rlen]

		reader := bufio.NewReader(bytes.NewBuffer(clippedRawBytes))

		oscMessage, _ := osc.ReadMessage(reader)

		connClient.Write(clippedRawBytes)

		//	fmt.Printf("received: %s from %s\n", oscMessage.Address, remote)

		if oscMessage.CountArguments() > 0 {

			bodypart, ok := oscMessage.Arguments[0].(string)

			if !ok {
				bodypart = "UNKNOWN of "+oscMessage.Address
			}

			posArray := osc.ReadFloatArguments(oscMessage, 1, 3)

			if len(posArray) == 3 {
				pos := math32.NewVector3(
					posArray[0],
					posArray[1],
					posArray[2],
				)

				fmt.Println(bodypart, pos)
			}
		}
	}

}
