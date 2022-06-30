package main

import (
	"fmt"
	"net"

	"github.com/rs/zerolog/log"
)

var rotationCheckBodyPartList []string = []string{
	"LeftUpperArm", "RightUpperArm",
	"LeftLowerArm", "RightLowerArm",
}

var config *Config

func main() {
	initilizeStuff()
	go startProfileKeyboardListener()

	_config, err := ReadConfig()
	config = _config

	if err != nil {
		log.Err(err).Msg("Some error")
		return
	}

	log.Info().Msgf("Logging the Differences of: %v", config.LogDiff)

	otherConnections, err := connectToOtherNodes(config)

	if err != nil {
		fmt.Printf("Some error %v", err)
		return
	}

	VmcListenerV1(otherConnections)
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
