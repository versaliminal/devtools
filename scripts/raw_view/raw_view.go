package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/h2non/filetype"
	"golang.org/x/term"
)

const (
	bytesPerRow = 16
)

type viewMode int

const (
	modeHexdump viewMode = iota
	modeLinear
	modeHilbert
)

type colorScheme int

const (
	schemeRanges colorScheme = iota
	schemePrintable
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <filename>\n", os.Args[0])
		os.Exit(1)
	}

	filename := os.Args[1]
	file, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// Get file size
	stat, err := file.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting file stats: %v\n", err)
		os.Exit(1)
	}

	fileSize := stat.Size()
	if fileSize == 0 {
		fmt.Println("File is empty")
		return
	}

	// Memory map the file
	data, err := syscall.Mmap(int(file.Fd()), 0, int(fileSize), syscall.PROT_READ, syscall.MAP_PRIVATE)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error memory mapping file: %v\n", err)
		os.Exit(1)
	}
	defer syscall.Munmap(data)

	// Set terminal to raw mode to capture keyboard input
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error setting raw mode: %v\n", err)
		os.Exit(1)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Hide cursor
	fmt.Print("\033[?25l")
	defer fmt.Print("\033[?25h")

	var offset int64 = 0
	var currentMode = modeHexdump
	var currentScheme = schemeRanges
	var lastMode = currentMode
	var lastScheme = currentScheme

	// Calculate global entropy once
	globalEntropy := calculateEntropy(data)

	for {
		// Get terminal dimensions
		width, height, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			width = 80
			height = 24
		}

		// Header lines:
		// 1-5: Color mapping (5 rows)
		// 6: Entropy info
		// 7: File/Mode info
		// 8: Empty
		// Final Row: Navigation help (at height)
		headerRows := 9
		displayRows := height - headerRows

		if displayRows < 1 {
			displayRows = 1
		}

		// Calculate Hilbert N for this screen size (Power of 2 fitting width)
		hilbertN := 1
		for hilbertN*2 <= width/2 {
			hilbertN *= 2
		}

		// Use a strings.Builder to buffer the entire screen output
		var screen strings.Builder

		// Clear screen when mode or scheme changes to avoid artifacts
		if currentMode != lastMode || currentScheme != lastScheme {
			screen.WriteString("\033[2J")
			lastMode = currentMode
			lastScheme = currentScheme
		}

		// Reset cursor to top-left
		screen.WriteString("\033[H")

		// Display header
		displayHeader(&screen, currentScheme)

		// Calculate visible bytes for entropy
		var visibleBytes int
		switch currentMode {
		case modeHexdump:
			visibleBytes = displayRows * bytesPerRow
		case modeLinear:
			visibleBytes = displayRows * (width / 2)
		case modeHilbert:
			visibleBytes = (displayRows / hilbertN) * (hilbertN * hilbertN)
			if visibleBytes == 0 {
				visibleBytes = hilbertN * hilbertN
			}
		}

		viewEnd := offset + int64(visibleBytes)
		if viewEnd > fileSize {
			viewEnd = fileSize
		}
		viewEntropy := calculateEntropy(data[offset:viewEnd])

		fmt.Fprintf(&screen, "Entropy: Global: %.4f bits/byte | View: %.4f bits/byte\033[K\r\n", globalEntropy, viewEntropy)

		modeName := ""
		switch currentMode {
		case modeHexdump:
			modeName = "Hexdump"
		case modeLinear:
			modeName = "Wrapped Linear"
		case modeHilbert:
			modeName = "Hilbert Curve"
		}

		fmt.Fprintf(&screen, "--- File: %s | Mode: %s | Offset: %08x / %08x ---\033[K\r\n\033[K\r\n", filename, modeName, offset, fileSize)

		switch currentMode {
		case modeHexdump:
			renderHexdump(&screen, data, fileSize, offset, displayRows, currentScheme)
		case modeLinear:
			renderLinear(&screen, data, fileSize, offset, width, displayRows, currentScheme)
		case modeHilbert:
			renderHilbert(&screen, data, fileSize, offset, hilbertN, displayRows, currentScheme)
		}

		// Clear the space between end of data and bottom navigation line
		screen.WriteString("\033[J")

		// Move to the bottom row for navigation help
		fmt.Fprintf(&screen, "\033[%d;1H", height)
		screen.WriteString("Nav: Arrows (Ln), PgUp/PgDn (Scr), Tab (Mode), / (Scheme), J (Jump), S (Search), Q (Exit)\033[K")

		// Output the entire buffered screen at once
		fmt.Print(screen.String())

		// Read key input
		buf := make([]byte, 8)
		n, err := os.Stdin.Read(buf)
		if err != nil && err != io.EOF {
			break
		}

		if n > 0 {
			key := buf[:n]
			// Check for 'q' or 'Q' or alone ESC
			if key[0] == 'q' || key[0] == 'Q' || (key[0] == 27 && n == 1) {
				break
			} else if key[0] == 's' || key[0] == 'S' {
				// Search for string
				fmt.Printf("\033[%d;1H\033[KSearch ASCII: ", height)
				term.Restore(int(os.Stdin.Fd()), oldState)

				scanner := bufio.NewScanner(os.Stdin)
				var searchStr string
				if scanner.Scan() {
					searchStr = scanner.Text()
				}

				if searchStr != "" {
					idx := bytes.Index(data, []byte(searchStr))
					if idx != -1 {
						offset = int64(idx)
						fmt.Printf("Found at offset: %08x. Press Enter to go...", idx)
						fmt.Scanln()
					} else {
						fmt.Printf("String not found. Press Enter to continue...")
						fmt.Scanln()
					}
				}

				// Re-enable raw mode
				oldState, _ = term.MakeRaw(int(os.Stdin.Fd()))
			} else if key[0] == 'j' || key[0] == 'J' {
				// Jump to offset
				fmt.Printf("\033[%d;1H\033[KJump to Hex Offset (e.g. 1A0): ", height)
				term.Restore(int(os.Stdin.Fd()), oldState)

				scanner := bufio.NewScanner(os.Stdin)
				var hexStr string
				if scanner.Scan() {
					hexStr = strings.TrimSpace(scanner.Text())
				}

				if hexStr != "" {
					newOffset, err := strconv.ParseInt(strings.TrimPrefix(hexStr, "0x"), 16, 64)
					if err == nil {
						offset = newOffset
					}
				}
				// Re-enable raw mode
				oldState, _ = term.MakeRaw(int(os.Stdin.Fd()))

			} else if key[0] == '/' {
				if currentScheme == schemeRanges {
					currentScheme = schemePrintable
				} else {
					currentScheme = schemeRanges
				}
			} else if key[0] == 9 { // Tab key
				currentMode = (currentMode + 1) % 3
			} else if key[0] == 27 && n > 1 { // Escape sequence
				if key[1] == '[' {
					if len(key) >= 3 {
						var step int64
						switch currentMode {
						case modeHexdump:
							step = int64(bytesPerRow)
						case modeLinear:
							step = int64(width / 2)
						case modeHilbert:
							step = int64(hilbertN)
						}

						switch key[2] {
						case 'A': // Up Arrow
							offset -= step
						case 'B': // Down Arrow
							offset += step
						case '5': // Page Up
							if len(key) >= 4 && key[3] == '~' {
								pageStep := step * int64(displayRows)
								if currentMode == modeHilbert {
									pageStep = int64(hilbertN * hilbertN)
								}
								offset -= pageStep
							}
						case '6': // Page Down
							if len(key) >= 4 && key[3] == '~' {
								pageStep := step * int64(displayRows)
								if currentMode == modeHilbert {
									pageStep = int64(hilbertN * hilbertN)
								}
								offset += pageStep
							}
						}
					}
				}
			}
		}

		// Boundary checks
		if offset < 0 {
			offset = 0
		}
		if fileSize > 0 {
			if offset >= fileSize {
				offset = ((fileSize - 1) / 16) * 16
			}
		} else {
			offset = 0
		}
	}

	// Final clear screen and reset cursor on exit
	fmt.Print("\033[2J\033[H")
	fmt.Println("Program exited.")
}

func renderHexdump(w io.Writer, data []byte, fileSize int64, offset int64, rows int, scheme colorScheme) {
	for i := 0; i < rows && (offset+int64(i*bytesPerRow)) < fileSize; i++ {
		rowOffset := offset + int64(i*bytesPerRow)
		fmt.Fprintf(w, "%08x: ", rowOffset)
		for j := 0; j < bytesPerRow; j++ {
			addr := rowOffset + int64(j)
			if addr < fileSize {
				val := data[addr]
				fmt.Fprintf(w, "%s%02x%s ", getColor(val, scheme), val, "\033[0m")
			} else {
				fmt.Fprint(w, "   ")
			}
		}
		fmt.Fprint(w, " | ")
		for j := 0; j < bytesPerRow; j++ {
			addr := rowOffset + int64(j)
			if addr < fileSize {
				val := data[addr]
				if val >= 32 && val <= 126 {
					fmt.Fprintf(w, "%c", val)
				} else {
					fmt.Fprint(w, ".")
				}
			} else {
				fmt.Fprint(w, " ")
			}
		}

		// Magic byte detection
		var magicInfo string
		if rowOffset < fileSize {
			// Get a chunk for detection (limit to something reasonable like 262 bytes or less)
			detectLen := 262
			if int64(detectLen) > fileSize-rowOffset {
				detectLen = int(fileSize - rowOffset)
			}
			kind, _ := filetype.Match(data[rowOffset : rowOffset+int64(detectLen)])
			if kind != filetype.Unknown {
				magicInfo = fmt.Sprintf(" | %s (%s)", kind.Extension, kind.MIME.Value)
			}
		}
		fmt.Fprintf(w, "%s\033[K\r\n", magicInfo)
	}
}

func renderLinear(w io.Writer, data []byte, fileSize int64, offset int64, width, rows int, scheme colorScheme) {
	bytesPerRowGrid := width / 2
	if bytesPerRowGrid < 1 {
		bytesPerRowGrid = 1
	}
	for i := 0; i < rows; i++ {
		for j := 0; j < bytesPerRowGrid; j++ {
			addr := offset + int64(i*bytesPerRowGrid) + int64(j)
			if addr < fileSize {
				val := data[addr]
				fmt.Fprintf(w, "%s  \033[0m", getColor(val, scheme))
			} else {
				break
			}
		}
		fmt.Fprint(w, "\033[K\r\n")
	}
}

func renderHilbert(w io.Writer, data []byte, fileSize int64, offset int64, n, displayRows int, scheme colorScheme) {
	currentOffset := offset
	rowsRemaining := displayRows

	for rowsRemaining > 0 && currentOffset < fileSize {
		// Calculate how many rows of this square we can show
		rowsToRender := n
		if rowsToRender > rowsRemaining {
			rowsToRender = rowsRemaining
		}

		// Prepare a grid for the square
		grid := make([][]byte, n)
		mask := make([][]bool, n)
		for i := range grid {
			grid[i] = make([]byte, n)
			mask[i] = make([]bool, n)
		}

		// Map indices to the grid
		for d := 0; d < n*n; d++ {
			var x, y int
			d2xy(n, d, &x, &y)
			addr := currentOffset + int64(d)
			if addr < fileSize {
				grid[y][x] = data[addr]
				mask[y][x] = true
			}
		}

		// Print the rows
		for y := 0; y < rowsToRender; y++ {
			for x := 0; x < n; x++ {
				if mask[y][x] {
					fmt.Fprintf(w, "%s  \033[0m", getColor(grid[y][x], scheme))
				} else {
					fmt.Fprint(w, "  ")
				}
			}
			fmt.Fprint(w, "\033[K\r\n")
		}

		rowsRemaining -= rowsToRender
		currentOffset += int64(n * n)
	}
}

func rot(n int, x, y *int, rx, ry int) {
	if ry == 0 {
		if rx == 1 {
			*x = n - 1 - *x
			*y = n - 1 - *y
		}
		*x, *y = *y, *x
	}
}

func d2xy(n int, d int, x, y *int) {
	var rx, ry, s, t = 0, 0, 0, d
	*x = 0
	*y = 0
	for s = 1; s < n; s *= 2 {
		rx = 1 & (t / 2)
		ry = 1 & (t ^ rx)
		rot(s, x, y, rx, ry)
		*x += s * rx
		*y += s * ry
		t /= 4
	}
}

func calculateEntropy(data []byte) float64 {
	if len(data) == 0 {
		return 0
	}
	counts := make([]int, 256)
	for _, b := range data {
		counts[b]++
	}
	var entropy float64
	for _, count := range counts {
		if count > 0 {
			p := float64(count) / float64(len(data))
			entropy -= p * math.Log2(p)
		}
	}
	return entropy
}

func displayHeader(w io.Writer, scheme colorScheme) {
	if scheme == schemeRanges {
		fmt.Fprint(w, "Color-coded byte viewer (Ranges):\033[K\r\n")
		fmt.Fprint(w, " 00-0F: \033[41m  \033[0m 10-1F: \033[42m  \033[0m 20-2F: \033[43m  \033[0m 30-3F: \033[44m  \033[0m\033[K\r\n")
		fmt.Fprint(w, " 40-4F: \033[45m  \033[0m 50-5F: \033[46m  \033[0m 60-6F: \033[47m  \033[0m 70-7F: \033[1;47m  \033[0m\033[K\r\n")
		fmt.Fprint(w, " 80-8F: \033[41m  \033[0m 90-9F: \033[42m  \033[0m A0-AF: \033[43m  \033[0m B0-BF: \033[44m  \033[0m\033[K\r\n")
		fmt.Fprint(w, " C0-CF: \033[45m  \033[0m D0-DF: \033[46m  \033[0m E0-EF: \033[47m  \033[0m F0-FF: \033[1;47m  \033[0m\033[K\r\n")
		fmt.Fprint(w, "\033[K\r\n")
	} else {
		fmt.Fprint(w, "Color-coded byte viewer (Printable):\033[K\r\n")
		fmt.Fprint(w, " Null: \033[40m  \033[0m Space: \033[44m  \033[0m Print: \033[42m  \033[0m Other: \033[41m  \033[0m\033[K\r\n")
		fmt.Fprint(w, "\033[K\r\n\033[K\r\n\033[K\r\n\033[K\r\n") // Keep header height consistent
	}
}

func getColor(value byte, scheme colorScheme) string {
	if scheme == schemePrintable {
		switch {
		case value == 0:
			return "\033[40m" // Null - Black
		case value == 32:
			return "\033[44m" // Space - Blue
		case value >= 33 && value <= 126:
			return "\033[42m" // Printable - Green
		default:
			return "\033[41m" // Non-printable - Red
		}
	}

	switch {
	case value <= 0x0F:
		return "\033[41m" // red
	case value <= 0x1F:
		return "\033[42m" // green
	case value <= 0x2F:
		return "\033[43m" // yellow
	case value <= 0x3F:
		return "\033[44m" // blue
	case value <= 0x4F:
		return "\033[45m" // magenta
	case value <= 0x5F:
		return "\033[46m" // cyan
	case value <= 0x6F:
		return "\033[47m" // light gray
	case value <= 0x7F:
		return "\033[1;47m" // white
	case value <= 0x8F:
		return "\033[41m" // red
	case value <= 0x9F:
		return "\033[42m" // green
	case value <= 0xAF:
		return "\033[43m" // yellow
	case value <= 0xBF:
		return "\033[44m" // blue
	case value <= 0xCF:
		return "\033[45m" // magenta
	case value <= 0xDF:
		return "\033[46m" // cyan
	case value <= 0xEF:
		return "\033[47m" // light gray
	default:
		return "\033[1;47m" // white
	}
}
