package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"

	"github.com/ungerik/go3d/fmath"
	go3dquat "github.com/ungerik/go3d/quaternion"
	vec3 "github.com/ungerik/go3d/vec3"
	vec4 "github.com/ungerik/go3d/vec4"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	// "slices"
	"strings"

	osc "just-do-it/osc"
)

// Go Cons:
// - no string concat template way
// - weird fixed array variable declarion syntax  var myArray[4]int

// maybe?
// - memory in fmt.println when called a bunch of times

var rotationCheckBodyPartList []string = []string{
	"LeftUpperArm", "RightUpperArm",
	"LeftLowerArm", "RightLowerArm",
}

var config *Config

func main() {
	initilizeStuff()

	_config, err := ReadConfig()
	config = _config

	if err != nil {
		log.Err(err).Msg("Some error")
		return
	}

	log.Info().Msgf("Logging the Differences of: %v", config.LogDiff)

	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		Port: int(config.ListenTo),
		IP:   net.ParseIP("0.0.0.0"),
	})
	if err != nil {
		log.Err(err).Msg("Cant listen to UDP Port")
		return
	}

	defer conn.Close()
	log.Info().Msgf("server listening %s", conn.LocalAddr().String())

	otherConnections, err := connectToOtherNodes(config)

	if err != nil {
		fmt.Printf("Some error %v", err)
		return
	}

	for {
		rawMessage := make([]byte, 2000)
		rlen, _, err := conn.ReadFromUDP(rawMessage[:])
		if err != nil {
			panic(err)
		}

		clippedRawBytes := rawMessage[0:rlen]

		filterAndSendToOthers(otherConnections, clippedRawBytes)
	}
}

func connectToOtherNodes(config *Config) ([]net.Conn, error) {
	amountOfOtherConnections := len(config.ReflectTo)

	otherConnections := make([]net.Conn, amountOfOtherConnections)

	for i := 0; i < amountOfOtherConnections; i++ {
		connClient, err := net.Dial("udp", config.ReflectTo[i])
		if err != nil {
			return nil, err
		}

		log.Info().Msgf("Sending Messages to " + config.ReflectTo[i])

		otherConnections[i] = connClient
	}

	return otherConnections, nil
}

var lastRotationOfPart = make(map[string]vec3.T)

func filterAndSendToOthers(connections []net.Conn, data []byte) {
	// this might be slow
	receivedDataAsString := string(data)

	if !strings.Contains(receivedDataAsString, "/VMC/Ext/Bone/Pos") {
		sendToAll(connections, data)
		return
	}

	foundBodyPartToCheck := false

	for _, bodyPartToCheck := range rotationCheckBodyPartList {
		foundBodyPartToCheck = strings.Contains(receivedDataAsString, bodyPartToCheck)

		if foundBodyPartToCheck {
			break
		}
	}

	if !foundBodyPartToCheck {
		sendToAll(connections, data)
		return
	}

	reader := bufio.NewReader(bytes.NewBuffer(data))

	oscMessage, _ := osc.ReadMessage(reader)

	if oscMessage.CountArguments() > 0 {

		bodypart := oscMessage.Arguments[0].(string)

		posArray := osc.ReadFloatArguments(oscMessage, 1, 3)

		quatArray := osc.ReadFloatArguments(oscMessage, 4, 7)

		if len(posArray) == 3 {
			pos := (*vec3.T)(posArray)

			fmt.Sprintln(bodypart, pos)

			quatVec4 := (*vec4.T)(quatArray)
			quat := go3dquat.FromVec4(quatVec4)
			quatEulerY, quatEulerX, quatEulerZ := quat.ToEulerAngles()

			quatEulerVec3 := vec3.T{quatEulerX, quatEulerY, quatEulerZ}

			// fmt.Printf("received: %s - %s - %v\n", oscMessage.Address, bodypart, quatEulerVec3)

			lastRotation := lastRotationOfPart[bodypart]

			if lastRotation[0] != 0 {

				if lastRotation[0] == quatEulerX &&
					lastRotation[1] == quatEulerY &&
					lastRotation[2] == quatEulerZ {
					// fmt.Printf("ITS THE SAME!!!: %v\n", bodypart)

					return
				}

				// fmt.Printf("lastPos: %s - %v\n", bodypart, lastRotation)

				diffX := fmath.Abs(lastRotation[0] - quatEulerX)
				diffY := fmath.Abs(lastRotation[1] - quatEulerY)
				diffZ := fmath.Abs(lastRotation[2] - quatEulerZ)

				maxChange := fmath.Max(diffX, fmath.Max(diffY, diffZ))

				if maxChange > config.LogDiff {
					log.Info().
						Dict("lastPos", zerolog.Dict().
							Float32("X", lastRotation[0]).
							Float32("Y", lastRotation[1]).
							Float32("Z", lastRotation[2]),
						).
						Dict("newPos", zerolog.Dict().
							Float32("X", quatEulerX).
							Float32("Y", quatEulerY).
							Float32("Z", quatEulerZ),
						).
						Dict("diff", zerolog.Dict().
							Float32("max", maxChange).
							Float32("X", diffX).
							Float32("Y", diffY).
							Float32("Z", diffZ),
						).Msg("Bodypart: " + bodypart)
				}

				sendToAll(connections, data)
			}

			lastRotationOfPart[bodypart] = quatEulerVec3
		}
	}
}

func sendToAll(connections []net.Conn, data []byte) {
	for _, conn := range connections {
		conn.Write(data)
	}
}
