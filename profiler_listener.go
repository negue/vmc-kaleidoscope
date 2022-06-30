package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"runtime/pprof"

	"github.com/eiannone/keyboard"

	"github.com/rs/zerolog/log"
)

var cpuProfilerRunning = false
var cpuProfFile *os.File

var err error

func startProfileKeyboardListener() {
	if err := keyboard.Open(); err != nil {
		panic(err)
	}
	defer func() {
		_ = keyboard.Close()
	}()

	fmt.Println("Press ESC to quit")
	for {
		char, key, err := keyboard.GetKey()
		if err != nil {
			panic(err)
		}
		fmt.Printf("You pressed: char %q, key %X\r\n", char, key)
		if key == keyboard.KeyEsc && char == '\x00' {
			fmt.Println("Closing the App")
			os.Exit(0)
		}

		if char == 'C' {
			toggleCpuProfiler()
		}

		if char == 'M' {
			triggerMemSnapshot()
		}
	}
}

func toggleCpuProfiler() {
	if cpuProfilerRunning {

		fmt.Println("Stopping the CPU Profiler")
		pprof.StopCPUProfile()
		cpuProfFile.Close()
		cpuProfFile = nil

		cpuProfilerRunning = false

		return
	}

	// Format("20060102150405") is a fixed date used as example for the format
	cpuProfFileName := "cpu_" + time.Now().Format("20060102150405") + ".prof"

	fmt.Println("Starting new CPU Profiler: " + cpuProfFileName)
	cpuProfFile, err = os.Create(cpuProfFileName)

	if err != nil {
		log.Err(err).Msg("Could not create / open " + cpuProfFileName)
		return
	}

	if err := pprof.StartCPUProfile(cpuProfFile); err != nil {
		log.Err(err).Msg("could not start CPU profile: ")
	}

	cpuProfilerRunning = true
}

func triggerMemSnapshot() {
	// Format("20060102150405") is a fixed date used as example for the format
	memProfFileName := "mem_" + time.Now().Format("20060102150405") + ".prof"

	fmt.Println("Starting new MEM Snapshot: " + memProfFileName)
	memProfFile, err := os.Create(memProfFileName)
	if err != nil {
		log.Err(err).Msg("Could not create / open " + memProfFileName)
		return
	}
	defer memProfFile.Close() // error handling omitted for example

	runtime.GC() // get up-to-date statistics

	if err := pprof.WriteHeapProfile(memProfFile); err != nil {
		log.Err(err).Msg("could not write memory profile: ")
	}
}
