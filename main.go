package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

var input *bufio.Reader
var graphLines [][]uint8

func nextChar() uint8 {
	ch, err := input.ReadByte()
	if err == io.EOF {
		fmt.Printf("\n%s\n", sprintGraph(graphLines))
		os.Exit(0)
	} else if err != nil {
		panic(err)
	}
	return ch
}

func nextNumber(size int) int {
	number := 0
	for i := 0; i < size; i++ {
		ch := nextChar()
		digit := int(ch - '0')
		number = number*10 + digit
	}
	return number
}

func nextList(size int) []int {
	list := make([]int, 0)
	for {
		list := append(list, nextNumber(size))
		separator := nextChar()
		switch separator {
		case '.':
			return list
		case ',':
			continue
		default:
			panic("Bad list format")
		}
	}
}

func nextBytes(count int) []uint8 {
	bytes := make([]uint8, count)
	for i := 0; i < count; i++ {
		bytes[i] = nextChar()
	}
	return bytes
}

type pattern struct {
	key  uint8
	top  bool
	data []uint8
}

func nextCharPattern(key uint8) *pattern {
	var p pattern
	p.key = key
	width := nextChar()
	if width >= 'a' {
		p.top = false
		width = width - 'a' + 1
	} else {
		p.top = true
		width = width - 'A' + 1
	}

	p.data = nextBytes(int(width))
	return &p
}

func nextCharPatterns() []*pattern {
	patterns := make([]*pattern, 0)
	for {
		key := nextChar()
		if key == 0x04 { // CTRL-D
			return patterns
		}
		p := nextCharPattern(key)
		patterns = append(patterns, p)
	}
}

// xterm -ti vt340
// mlterm
func sprintPattern(p *pattern) string {
	var s0 strings.Builder
	var s1 strings.Builder
	for _, col := range p.data {
		bits := uint16(col)
		if !p.top {
			bits <<= 1
		}
		b0 := (bits & 0x3f) + 63
		b1 := (bits >> 6) + 63
		s0.WriteByte(uint8(b0))
		s1.WriteByte(uint8(b1))
	}
	return fmt.Sprintf("\x1bPq%s-%s\x1b\\", s0.String(), s1.String())
}

func sprintGraphRow(data []uint8) string {
	var s0 strings.Builder
	var s1 strings.Builder

	for _, col := range data {
		b0 := (col & 0x3f) + 63
		b1 := (col >> 6) + 63
		s0.WriteByte(b0)
		s1.WriteByte(b1)
	}
	return fmt.Sprintf("\x1bPq%s-%s\x1b\\", s0.String(), s1.String())
}

func sprintGraph(data [][]uint8) string {
	var sb strings.Builder

	// Ensure len is multiple of 6
	for len(data)%6 != 0 {
		data = append(data, make([]uint8, 0))
	}

	width := 0
	for _, row := range data {
		if len(row) > width {
			width = len(row)
		}
	}
	height := len(data) * 8

	for i := 0; i < height/(6*8); i++ {
		// Build row grouping 6 bytes
		row := make([]uint64, width)
		for x := 0; x < width; x++ {
			fatBits := uint64(0)
			for j := 5; j >= 0; j-- {
				rowBytes := data[i*6+j]
				b := uint8(0)
				if x < len(rowBytes) {
					b = rowBytes[x]
				}
				fatBits = (fatBits << 8) + uint64(b)
			}
			row[x] = fatBits
		}

		// Print row in 8 groups
		for j := 0; j < 8; j++ {
			for x := 0; x < width; x++ {
				b := (row[x] & 0x3f) + 63
				row[x] = row[x] >> 6
				sb.WriteByte(uint8(b))
			}
			sb.WriteByte('-') // New graphic line
		}
	}

	return fmt.Sprintf("\x1bPq%s\x1b\\", sb.String())
}

func readEscapeSequence() string {
	s := "Unknown"
	command := nextChar()
	switch command {

	// Print Quality
	case 'a':
		mode := nextNumber(1)
		s = fmt.Sprintf("Print quality %v", mode)
	case 'm':
		s = "Print quality 0-correspondence"
	case 'M':
		s = "Print quality 2-near-letter"

	// Software Switch Commands
	case 'Z':
		bits := (uint16(nextChar()) << 8) + uint16(nextChar())
		s = fmt.Sprintf("Open switches %016b", bits)
	case 'D':
		bits := (uint16(nextChar()) << 8) + uint16(nextChar())
		s = fmt.Sprintf("Close switches %016b", bits)

	// User-Designed Characters
	case '-':
		s = "Max width of custom chars to 8 dots"
	case '+':
		s = "Max width of custom chars to 16 dots"
	case 'I':
		patterns := nextCharPatterns()
		s = fmt.Sprintf("Load %d new characters", len(patterns))
		for _, p := range patterns {
			s = fmt.Sprintf("%s\n    Key '%c', %v bytes", s, p.key, len(p.data))
			s += sprintPattern(p)
		}
	case '\'':
		s = "Switch to custom character font"
	case '*':
		s = "Switch to custom character font (high ASCII)"
	case '$':
		s = "Switch to normal font"
	case '&':
		s = "Map MouseText to low ASCII"

	// Character Pitch
	case 'n':
		s = "Pitch 9 cpi"
	case 'N':
		s = "Pitch 10 cpi"
	case 'E':
		s = "Pitch 12 cpi"
	case 'e':
		s = "Pitch 13.4 cpi"
	case 'q':
		s = "Pitch 15 cpi"
	case 'Q':
		s = "Pitch 17 cpi"
	case 'p':
		s = "Pitch 144 dpi"
	case 'P':
		s = "Pitch 160 dpi"

	// Proportional Character Spacing
	case 's':
		dotSpacing := nextNumber(1)
		s = fmt.Sprintf("Dot spacing to %v", dotSpacing)
	case '1', '2', '3', '4', '5', '6':
		s = fmt.Sprintf("Insert %c dot spaces", command)

	// Character Attributes
	case 'X':
		s = "Start underline"
	case 'Y':
		s = "Stop underline"
	case '!':
		s = "Start bold"
	case '"':
		s = "Stop bold"
	case 'w':
		s = "Start half-height"
	case 'W':
		s = "Stop half-height"
	case 'x':
		s = "Start superscript"
	case 'y':
		s = "Start subscript"
	case 'z':
		s = "Stop superscript or subscript"

	// Page formatting
	case 'L':
		column := nextNumber(3)
		s = fmt.Sprintf("Set left margin at column %v", column)
	case 'H':
		length := nextNumber(4) // In 1/144 of an inch
		s = fmt.Sprintf("Set page length to %v/144 inches", length)

	// Print Head Motion
	case '>':
		s = "Unidirectional printing"
	case '<':
		s = "Bidirectional printing"
	case '(':
		tabs := nextList(3)
		s = fmt.Sprintf("Set tabs at %v", tabs)
	case 'u':
		column := nextNumber(3)
		s = fmt.Sprintf("Add tab at column %v", column)
	case ')':
		tabs := nextList(3)
		s = fmt.Sprintf("Clear tabs at %v", tabs)
	case '0':
		s = "Clear all tabs"
	case 'F':
		pos := nextNumber(4)
		s = fmt.Sprintf("Place print head at pixel position %v", pos)

	// Paper motion
	case 'v':
		s = "Set top of file"
	case 'A':
		s = "6 lines per inch"
	case 'B':
		s = "8 lines per inch"
	case 'T':
		interline := nextNumber(2) // In 1/144 of an inch
		s = fmt.Sprintf("Distance between lines %v/144 inches", interline)
	case 'f':
		s = "Forward line feeding"
	case 'r':
		s = "Reverse line feeding"
	case 'O':
		s = "Paper-out sensor off"
	case 'o':
		s = "Paper-out sensor on"

	// Automatic CR and LF
	case 'l':
		mode := nextNumber(1)
		switch mode {
		case '0':
			s = "No CR insertion before LF and FF"
		case '1':
			s = "Insert CR before LF and DD"
		default:
			s = "CR insertion undefined"
		}

	// Graphic control
	case 'G', 'S':
		count := nextNumber(4)
		bytes := nextBytes(count)
		s = fmt.Sprintf("Graphic line, %v bytes%s", len(bytes), sprintGraphRow(bytes))
		graphLines = append(graphLines, bytes)
	case 'g':
		count := 8 * nextNumber(3)
		bytes := nextBytes(count)
		s = fmt.Sprintf("Graphic line, %v bytes%s", len(bytes), sprintGraphRow(bytes))
		graphLines = append(graphLines, bytes)
	case 'V':
		count := nextNumber(4)
		pattern := nextChar()
		bytes := make([]uint8, count)
		for i := 0; i < count; i++ {
			bytes[i] = pattern
		}
		s = fmt.Sprintf("Graphic line, %v times%s", len(bytes), sprintGraphRow(bytes))
		graphLines = append(graphLines, bytes)

	// Color printing
	case 'K':
		color := nextNumber(1)
		s = fmt.Sprintf("Set color %v", color)

	// Miscellaneous commands
	case 'R':
		count := nextNumber(3)
		ch := nextChar()
		s = fmt.Sprintf("Repeat char '%c', %v times", ch, count)
	case 'c':
		s = "Reset defaults"
	case '?':
		s = "Send ID string"
	}

	return fmt.Sprintf("<Escape %c:\"%s\">", command, s)
}

func main() {
	input = bufio.NewReader(os.Stdin)
	graphLines = make([][]uint8, 0)

	for {
		ch := nextChar()
		switch {
		case ch == 0x1b: // ESCAPE
			s := readEscapeSequence()
			fmt.Printf("\n%s", s)
		case ch < 0x20: // Control codes
			s := ""
			switch ch {
			case 0x0e: // CTRL-N
				s = " Double width"
			case 0x0f: // CTRL-O
				s = " Single width"
			}
			fmt.Printf("\n{%02x:^%c%s}", ch, ch+'@', s)
		default:
			fmt.Printf("{%02x}", ch)
		}
	}
}
