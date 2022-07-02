package main

import (
	"fmt"
	"net"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ungerik/go3d/fmath"
	go3dquat "github.com/ungerik/go3d/quaternion"
	vec3 "github.com/ungerik/go3d/vec3"

	"errors"

	"github.com/dnaka91/go-vmcparser/osc"
	"github.com/dnaka91/go-vmcparser/vmc"
)

func VmcListenerV2(connections []net.Conn) {
	conn, err := net.ListenPacket("udp", fmt.Sprintf(":%v", config.ListenTo))
	if err != nil {
		log.Err(err).Msg("Cant listen to UDP Port")
		return
	}

	defer conn.Close()
	log.Info().Msgf("server listening %s", conn.LocalAddr().String())

	// Create a new UDP listener at the VMC default port.

	// Create a new buffer to read UPD payloads into.
	//
	// The value 1536 is a common maximum size for Ethernet II, meaning we only need
	// a single (or few) system calls to get the content from the OS handlers.
	//
	// Pick any other value as you like ;-)
	buf := make([]byte, 1536)

	var lastRotationOfPartV2 = make(map[string]vec3.T)

	// Start an endless loop, trying to get messages until the end of time, or some error happens.
	for {
		// Read a new packet from the connection.
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			log.Err(err).Msgf("failed to read from the UDP connection: %v", err)
		}

		filledBuffer := buf[:n]

		// Fail if we got an OSC bundle, instead of a single OSC message.
		if osc.IsBundle(filledBuffer) {
			log.Fatal().Msg("got an OSC bundle, we don't handle them (yet)")
		}

		// Parse the message into a known VMC message.
		//
		// Here we pass some extra filters, so we'll only fully parse
		// the root and bone transform messages (for best possible performance).
		message, err := vmc.ParseMessage(
			buf,
			// vmc.AddressRootTransform,
			vmc.AddressBoneTransform,
		)

		if errors.Is(err, vmc.ErrFiltered) {
			// not one of the VMC Messages that needs to be filtered, so just push them out
			sendToAll(connections, filledBuffer)
			continue
		}

		if err != nil {
			log.Fatal().Msgf("failed to parse VMC message: %v", err)
		}

		// Finally we got our message parsed and ready. Now we can cast it into
		// one of the several defined messages and access their content as proper
		// Go structs.
		switch m := message.(type) {
		case *vmc.BoneTransform:
			log.Printf("new root transformation, named  %v with position %v\n", m.Name, m.Position)

			currentBodyPart := string(m.Name)

			foundBodyPartToCheck := false

			for _, bodyPartToCheck := range rotationCheckBodyPartList {
				foundBodyPartToCheck = strings.Contains(currentBodyPart, bodyPartToCheck)

				if foundBodyPartToCheck {
					break
				}
			}

			if !foundBodyPartToCheck {
				// none of the bodyparts to be filtered, so push them out
				sendToAll(connections, filledBuffer)
				continue
			}

			// pos := vec3.T{m.Position.X, m.Position.Y, m.Position.Z}
			quat := go3dquat.T{m.Quaternion.X, m.Quaternion.Y, m.Quaternion.Z, m.Quaternion.W}

			quatEulerY, quatEulerX, quatEulerZ := quat.ToEulerAngles()

			quatEulerVec3 := vec3.T{quatEulerX, quatEulerY, quatEulerZ}

			// fmt.Printf("received: %s - %s - %v\n", oscMessage.Address, bodypart, quatEulerVec3)

			// TODO to be extracted once the actual rotation/position filtering is added

			lastRotation := lastRotationOfPartV2[currentBodyPart]

			if lastRotation[0] != 0 {

				if lastRotation[0] == quatEulerX &&
					lastRotation[1] == quatEulerY &&
					lastRotation[2] == quatEulerZ {
					// fmt.Printf("ITS THE SAME!!!: %v\n", bodypart)

					continue
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
						).Msg("Bodypart: " + currentBodyPart)
				}

				sendToAll(connections, filledBuffer)
			}

			lastRotationOfPartV2[currentBodyPart] = quatEulerVec3

		default:
			log.Printf("got message from %v: %v\n", addr, message)
		}
	}
}

func sendToAll(connections []net.Conn, data []byte) {
	for _, conn := range connections {
		conn.Write(data)
	}
}
