package main

import (
	"fmt"
	"math"
	"math/bits"
)

/*
The codes of the Epson printers, as the FX-80, which many programs of the
Apple II print with. The graphics put the top dot in the most significant bit
of each byte, and the paper moves in 1/72 or 1/216 of an inch.
*/

// epsonDotsPerInch of the graphics for each mode of ESC *
var epsonDotsPerInch = map[uint8]float64{
	0: 60,
	1: 120,
	2: 120,
	3: 240,
	4: 80,
	5: 72,
	6: 90,
	7: 144,
}

// epsonColors are the colors of ESC r as the colors of ESC K of the
// ImageWriter: black, magenta, cyan, violet, yellow, orange and green
var epsonColors = []int{0, 2, 3, 6, 1, 4, 5}

func nextWord() int {
	low := int(nextChar())
	return low + int(nextChar())<<8
}

// skipUntilNul skips the list of a command, as the tabs of ESC D
func skipUntilNul() int {
	count := 0
	for nextChar() != 0 {
		count++
	}
	return count
}

// epsonGraphics prints a line of graphics, the bytes turned around so that
// the top dot is in the least significant bit, as the ImageWriter has it
func epsonGraphics(dpi float64, count int) string {
	columns := nextBytes(count)
	for i, column := range columns {
		columns[i] = bits.Reverse8(column)
	}
	page.dpi = dpi
	page.graphics(columns)
	graphLines = append(graphLines, columns)
	return fmt.Sprintf("Graphic line at %v dpi, %v bytes%s", dpi, count, sprintGraphRow(columns))
}

func readEpsonEscapeSequence() string {
	s := "Unknown"
	command := nextChar()
	switch command {

	// Graphics
	case 'K':
		s = epsonGraphics(60, nextWord())
	case 'L', 'Y':
		s = epsonGraphics(120, nextWord())
	case 'Z':
		s = epsonGraphics(240, nextWord())
	case '*':
		mode := nextChar()
		count := nextWord()
		if dpi, ok := epsonDotsPerInch[mode]; ok {
			s = epsonGraphics(dpi, count)
		} else {
			nextBytes(count)
			s = fmt.Sprintf("Graphic line of unknown mode %v, %v bytes", mode, count)
		}
	case '^':
		mode := nextChar()
		count := nextWord()
		nextBytes(2 * count)
		s = fmt.Sprintf("Graphic line of 9 dots, mode %v, %v columns", mode, count)

	// Paper motion
	case '0':
		page.lineDistance = 18
		s = "Distance between lines 1/8 inches"
	case '1':
		page.lineDistance = 14
		s = "Distance between lines 7/72 inches"
	case '2':
		page.lineDistance = 24
		s = "Distance between lines 1/6 inches"
	case '3':
		n := nextChar()
		page.lineDistance = float64(n) * 144 / 216
		s = fmt.Sprintf("Distance between lines %v/216 inches", n)
	case 'A':
		n := nextChar()
		page.lineDistance = float64(n) * 2
		s = fmt.Sprintf("Distance between lines %v/72 inches", n)
	case 'J':
		n := nextChar()
		page.y += float64(n) * 144 / 216
		s = fmt.Sprintf("Feed %v/216 inches", n)
	case 'j':
		n := nextChar()
		page.y = math.Max(page.y-float64(n)*144/216, 0)
		s = fmt.Sprintf("Reverse feed %v/216 inches", n)
	case 'C':
		lines := nextChar()
		if lines == 0 {
			inches := nextChar()
			page.pageLength = float64(inches) * 144
			s = fmt.Sprintf("Set page length to %v inches", inches)
		} else {
			page.pageLength = float64(lines) * page.lineDistance
			s = fmt.Sprintf("Set page length to %v lines", lines)
		}
	case 'N':
		lines := nextChar()
		s = fmt.Sprintf("Skip %v lines over the perforation", lines)
	case 'O':
		s = "No skip over the perforation"

	// Print head motion
	case '$':
		n := nextWord()
		page.x = float64(n) / 60
		s = fmt.Sprintf("Place print head at %v/60 inches", n)
	case '\\':
		n := int16(nextWord())
		page.x = math.Max(page.x+float64(n)/120, 0)
		s = fmt.Sprintf("Move print head %v/120 inches", n)
	case 'D':
		s = fmt.Sprintf("Set %v tabs", skipUntilNul())
	case 'B':
		s = fmt.Sprintf("Set %v vertical tabs", skipUntilNul())
	case '<':
		s = "Unidirectional printing for one line"
	case 'U':
		n := nextChar()
		s = fmt.Sprintf("Unidirectional printing %v", n)

	// Color printing
	case 'r':
		color := int(nextChar())
		if color < len(epsonColors) {
			page.setColor(epsonColors[color])
		}
		s = fmt.Sprintf("Set color %v", color)

	// User-defined characters
	case '&':
		nextChar()
		first := int(nextChar())
		last := int(nextChar())
		for ch := first; ch <= last; ch++ {
			nextBytes(12) // The attribute and 11 columns
		}
		s = fmt.Sprintf("Load %v new characters", last-first+1)
	case ':':
		nextBytes(3)
		s = "Copy the font to the user-defined characters"

	// Commands with a parameter, of the characters and the margins
	case '!', '-', 'W', 'S', 'Q', 'l', 'x', 'k', 'p', 'I', 'R', 't', 's',
		'%', 'a', 'i', ' ', 0x19:
		n := nextChar()
		s = fmt.Sprintf("Character setting with %v", n)

	// Commands without parameters
	case 'E', 'F', 'G', 'H', '4', '5', 'M', 'P', 'T', '8', '9', '6', '7',
		'#', '=', '>', 0x0e, 0x0f:
		s = "Character setting"
	case '@':
		page.lineDistance = 24
		page.setColor(0)
		s = "Reset defaults"
	}

	return fmt.Sprintf("<Escape %c:\"%s\">", command, s)
}

func epson() {
	for {
		ch := nextChar()
		switch {
		case ch == 0x1b: // ESCAPE
			s := readEpsonEscapeSequence()
			fmt.Printf("\n%s", s)
		case ch < 0x20: // Control codes
			switch ch {
			case 0x0a: // CTRL-J
				page.lineFeed()
			case 0x0c: // CTRL-L
				page.formFeed()
			case 0x0d: // CTRL-M
				page.carriageReturn()
			}
			fmt.Printf("\n{%02x:^%c}", ch, ch+'@')
		default:
			fmt.Printf("{%02x}", ch)
		}
	}
}
