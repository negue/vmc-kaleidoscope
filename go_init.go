package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	// "slices"
	"strings"
)

// copied from zerolog
const (
	colorBlack = iota + 30
	colorRed
	colorGreen
	colorYellow
	colorBlue
	colorMagenta
	colorCyan
	colorWhite

	colorBold     = 1
	colorDarkGray = 90
)

func initilizeStuff() {
	rand.Seed(time.Now().UnixNano())

	logFile, _ := os.OpenFile("./osc.log", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)

	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}

	consoleWriter.FormatLevel = func(i interface{}) string {
		var l string
		if ll, ok := i.(string); ok {
			switch ll {
			case zerolog.LevelTraceValue:
				l = colorize("TRACE", colorMagenta)
			case zerolog.LevelDebugValue:
				l = colorize("DEBUG", colorYellow)
			case zerolog.LevelInfoValue:
				l = colorize("INFO", colorGreen)
			case zerolog.LevelWarnValue:
				l = colorize("WARN", colorRed)
			case zerolog.LevelErrorValue:
				l = colorize(colorize("ERROR", colorRed), colorBold)
			case zerolog.LevelFatalValue:
				l = colorize(colorize("FATAL", colorRed), colorBold)
			case zerolog.LevelPanicValue:
				l = colorize(colorize("PANIC", colorRed), colorBold)
			default:
				l = colorize("???", colorBold)
			}
		} else {
			if i == nil {
				l = colorize("???", colorBold)
			} else {
				l = strings.ToUpper(fmt.Sprintf("%s", i))[0:3]
			}
		}
		return fmt.Sprintf("| %-8s |", l)
	}

	w := zerolog.MultiLevelWriter(logFile, consoleWriter)

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(w)
}

func colorize(s interface{}, c int) string {
	return fmt.Sprintf("\x1b[%dm%v\x1b[0m", c, s)
}
