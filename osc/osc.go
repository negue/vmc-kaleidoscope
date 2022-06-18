package osc

// copied the basic UPD Message Parsing from github.com/hypebeast/go-osc
// since the full package was to slow (and to demanding on the CPU/ram)

import (
	"bufio"
	"encoding/binary"
	"fmt"
)

type Message struct {
	Address   string
	Arguments []interface{}
}

func NewMessage(addr string, args ...interface{}) *Message {
	return &Message{Address: addr, Arguments: args}
}

// Append appends the given arguments to the arguments list.
func (msg *Message) Append(args ...interface{}) {
	msg.Arguments = append(msg.Arguments, args...)
}

// CountArguments returns the number of arguments.
func (msg *Message) CountArguments() int {
	return len(msg.Arguments)
}

func ReadMessage(reader *bufio.Reader) (*Message, error) {
	// First, read the OSC address
	addr, _, err := readPaddedString(reader)
	if err != nil {
		return nil, err
	}

	// Read all arguments
	msg := NewMessage(addr)
	if err = readArguments(msg, reader); err != nil {
		return nil, err
	}

	return msg, nil
}

func readArguments(msg *Message, reader *bufio.Reader) error {
	// Read the type tag string
	typetags, _, err := readPaddedString(reader)
	if err != nil {
		return err
	}

	if len(typetags) == 0 {
		return nil
	}

	// If the typetag doesn't start with ',', it's not valid
	if typetags[0] != ',' {
		return fmt.Errorf("unsupported type tag string %s", typetags)
	}

	// Remove ',' from the type tag
	typetags = typetags[1:]

	for _, c := range typetags {
		switch c {
		default:
			return fmt.Errorf("unsupported type tag: %c", c)

		case 'i': // int32
			var i int32
			if err = binary.Read(reader, binary.BigEndian, &i); err != nil {
				return err
			}
			msg.Append(i)

		case 'h': // int64
			var i int64
			if err = binary.Read(reader, binary.BigEndian, &i); err != nil {
				return err
			}
			msg.Append(i)

		case 'f': // float32
			var f float32
			if err = binary.Read(reader, binary.BigEndian, &f); err != nil {
				return err
			}
			msg.Append(f)

		case 'd': // float64/double
			var d float64
			if err = binary.Read(reader, binary.BigEndian, &d); err != nil {
				return err
			}
			msg.Append(d)

		case 's': // string
			// TODO: fix reading string value
			var s string
			if s, _, err = readPaddedString(reader); err != nil {
				return err
			}
			msg.Append(s)
		}
	}

	return nil
}

func readPaddedString(reader *bufio.Reader) (string, int, error) {
	// Read the string from the reader
	str, err := reader.ReadString(0)
	if err != nil {
		return "", 0, err
	}
	n := len(str)

	// Remove the padding bytes (leaving the null delimiter)
	padLen := padBytesNeeded(len(str))
	if padLen > 0 {
		n += padLen
		padBytes := make([]byte, padLen)
		if _, err = reader.Read(padBytes); err != nil {
			return "", 0, err
		}
	}

	// Strip off the string delimiter
	return str[:len(str)-1], n, nil
}

func padBytesNeeded(elementLen int) int {
	return ((4 - (elementLen % 4)) % 4)
}

func ReadFloatArguments(msg *Message, startPos int, endPos int) []float32 {
	result := make([]float32, endPos-startPos+1)

	if msg.CountArguments() < endPos {
		return result
	}

	pos := 0
	for i := startPos; i <= endPos; i++ {
		curVal := msg.Arguments[i]

		switch typedValue := curVal.(type) {
		case float32:
			{
				result[pos] = typedValue
				pos++
			}
		}
	}

	return result
}
