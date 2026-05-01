package desktopbroker

import (
	"bytes"
	"fmt"
	"strconv"
)

type Decoder struct {
	buffer bytes.Buffer
}

func (d *Decoder) Feed(chunk []byte) ([][]string, error) {
	if len(chunk) > 0 {
		if _, err := d.buffer.Write(chunk); err != nil {
			return nil, err
		}
	}

	var instructions [][]string
	data := d.buffer.Bytes()
	cursor := 0

	for {
		start := cursor
		parts := make([]string, 0, 4)

		for {
			lengthStart := cursor
			for cursor < len(data) && data[cursor] >= '0' && data[cursor] <= '9' {
				cursor++
			}
			if cursor == len(data) {
				cursor = start
				d.discard(cursor)
				return instructions, nil
			}
			if cursor == lengthStart || data[cursor] != '.' {
				return instructions, fmt.Errorf("invalid instruction length at offset %d", cursor)
			}

			partLength, err := strconv.Atoi(string(data[lengthStart:cursor]))
			if err != nil {
				return instructions, fmt.Errorf("parse instruction length: %w", err)
			}
			cursor++

			if cursor+partLength > len(data) {
				cursor = start
				d.discard(cursor)
				return instructions, nil
			}

			parts = append(parts, string(data[cursor:cursor+partLength]))
			cursor += partLength

			if cursor == len(data) {
				cursor = start
				d.discard(cursor)
				return instructions, nil
			}

			separator := data[cursor]
			cursor++

			switch separator {
			case ',':
				continue
			case ';':
				instructions = append(instructions, parts)
			default:
				return instructions, fmt.Errorf("invalid instruction separator %q", separator)
			}
			break
		}

		if cursor >= len(data) {
			d.discard(cursor)
			return instructions, nil
		}
	}
}

func (d *Decoder) discard(n int) {
	if n <= 0 {
		return
	}
	remaining := append([]byte(nil), d.buffer.Bytes()[n:]...)
	d.buffer.Reset()
	if len(remaining) > 0 {
		_, _ = d.buffer.Write(remaining)
	}
}

func EncodeInstruction(parts ...string) string {
	var buffer bytes.Buffer
	for index, part := range parts {
		if index > 0 {
			buffer.WriteByte(',')
		}
		buffer.WriteString(strconv.Itoa(len(part)))
		buffer.WriteByte('.')
		buffer.WriteString(part)
	}
	buffer.WriteByte(';')
	return buffer.String()
}
