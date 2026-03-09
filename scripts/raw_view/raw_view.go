package main

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/h2non/filetype"
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
	scheme256Colors
)

type model struct {
	data          []byte
	fileSize      int64
	filename      string
	offset        int64
	currentMode   viewMode
	currentScheme colorScheme
	width         int
	height        int
	globalEntropy float64

	// Search/Jump state
	searching     bool
	jumping       bool
	textInput     textinput.Model
	searchMsg     string
	pendingOffset int64
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if m.searching || m.jumping {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				if m.searching {
					searchStr := m.textInput.Value()
					if searchStr != "" {
						idx := bytes.Index(m.data, []byte(searchStr))
						if idx != -1 {
							m.pendingOffset = int64(idx)
							m.searchMsg = fmt.Sprintf("Found at offset: %08x. Press Enter to go.", idx)
						} else {
							m.searchMsg = "String not found."
							m.pendingOffset = -1
						}
					}
					m.searching = false
				} else if m.jumping {
					hexStr := strings.TrimSpace(m.textInput.Value())
					if hexStr != "" {
						newOffset, err := strconv.ParseInt(strings.TrimPrefix(hexStr, "0x"), 16, 64)
						if err == nil {
							m.pendingOffset = newOffset
							m.searchMsg = fmt.Sprintf("Jump to offset: %08x. Press Enter to go.", newOffset)
						} else {
							m.searchMsg = "Invalid hex offset."
							m.pendingOffset = -1
						}
					}
					m.jumping = false
				}
				m.textInput.Blur()
				return m, nil
			case "esc":
				m.searching = false
				m.jumping = false
				m.textInput.Blur()
				return m, nil
			}
		}
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// If we are showing a message, clear it.
		if m.searchMsg != "" {
			if msg.String() == "enter" && m.pendingOffset != -1 {
				m.offset = m.pendingOffset
			}
			m.searchMsg = ""
			m.pendingOffset = -1
			// Fall through to allow 's', 'j', and other keys to work immediately
		}

		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit

		case "tab":
			m.currentMode = (m.currentMode + 1) % 3

		case "/":
			if m.currentScheme == schemeRanges {
				m.currentScheme = schemePrintable
			} else if m.currentScheme == schemePrintable {
				m.currentScheme = scheme256Colors
			} else {
				m.currentScheme = schemeRanges
			}

		case "s", "S":
			m.searching = true
			m.jumping = false
			m.textInput.Focus()
			m.textInput.SetValue("")
			m.textInput.Prompt = "Search ASCII: "
			m.searchMsg = ""
			m.pendingOffset = -1
			return m, nil

		case "j", "J":
			m.jumping = true
			m.searching = false
			m.textInput.Focus()
			m.textInput.SetValue("")
			m.textInput.Prompt = "Jump to Hex Offset (e.g. 1A0): "
			m.searchMsg = ""
			m.pendingOffset = -1
			return m, nil

		case "up":
			m.offset -= m.getStep()
		case "down":
			m.offset += m.getStep()
		case "pgup":
			m.offset -= m.getPageStep()
		case "pgdown":
			m.offset += m.getPageStep()
		}
	}

	// Boundary checks
	if m.offset < 0 {
		m.offset = 0
	}
	if m.fileSize > 0 {
		if m.offset >= m.fileSize {
			m.offset = ((m.fileSize - 1) / 16) * 16
		}
	} else {
		m.offset = 0
	}

	return m, nil
}

func (m model) getStep() int64 {
	switch m.currentMode {
	case modeHexdump:
		return int64(bytesPerRow)
	case modeLinear:
		return int64(m.width / 2)
	case modeHilbert:
		return int64(m.getHilbertN())
	default:
		return 1
	}
}

func (m model) getPageStep() int64 {
	headerRows := 9
	displayRows := m.height - headerRows
	if displayRows < 1 {
		displayRows = 1
	}

	switch m.currentMode {
	case modeHilbert:
		n := m.getHilbertN()
		return int64(n * n)
	default:
		return m.getStep() * int64(displayRows)
	}
}

func (m model) getHilbertN() int {
	hilbertN := 1
	for hilbertN*2 <= m.width/2 {
		hilbertN *= 2
	}
	return hilbertN
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	headerRows := 10 // Reserve space for header and footer
	displayRows := m.height - headerRows
	if displayRows < 1 {
		displayRows = 1
	}

	hilbertN := m.getHilbertN()

	var lines []string

	// 1. Display header (6 lines)
	headerLines := getHeaderLines(m.currentScheme)
	lines = append(lines, headerLines...)

	// 2. Entropy line (1 line)
	var visibleBytes int
	switch m.currentMode {
	case modeHexdump:
		visibleBytes = displayRows * bytesPerRow
	case modeLinear:
		visibleBytes = displayRows * (m.width / 2)
	case modeHilbert:
		visibleBytes = (displayRows / hilbertN) * (hilbertN * hilbertN)
		if visibleBytes == 0 {
			visibleBytes = hilbertN * hilbertN
		}
	}

	viewEnd := m.offset + int64(visibleBytes)
	if viewEnd > m.fileSize {
		viewEnd = m.fileSize
	}
	viewEntropy := calculateEntropy(m.data[m.offset:viewEnd])
	lines = append(lines, fmt.Sprintf("Entropy: Global: %.4f bits/byte | View: %.4f bits/byte", m.globalEntropy, viewEntropy))

	// 3. Mode/File line (1 line)
	modeName := ""
	switch m.currentMode {
	case modeHexdump:
		modeName = "Hexdump"
	case modeLinear:
		modeName = "Wrapped Linear"
	case modeHilbert:
		modeName = "Hilbert Curve"
	}
	lines = append(lines, fmt.Sprintf("--- File: %s | Mode: %s | Offset: %08x / %08x ---", m.filename, modeName, m.offset, m.fileSize))

	// 4. Spacer line (1 line)
	lines = append(lines, "")

	// 5. Data rows (displayRows)
	var dataBuf strings.Builder
	switch m.currentMode {
	case modeHexdump:
		renderHexdump(&dataBuf, m.data, m.fileSize, m.offset, displayRows, m.currentScheme)
	case modeLinear:
		renderLinear(&dataBuf, m.data, m.fileSize, m.offset, m.width, displayRows, m.currentScheme)
	case modeHilbert:
		renderHilbert(&dataBuf, m.data, m.fileSize, m.offset, hilbertN, displayRows, m.currentScheme)
	}
	dataLines := strings.Split(strings.TrimSuffix(dataBuf.String(), "\n"), "\n")
	for i := 0; i < displayRows && i < len(dataLines); i++ {
		lines = append(lines, dataLines[i])
	}

	// 6. Padding to reach the footer (height - 1)
	for len(lines) < m.height-1 {
		lines = append(lines, "")
	}

	// 7. Footer (1 line)
	footer := ""
	if m.searching || m.jumping {
		footer = m.textInput.View()
	} else if m.searchMsg != "" {
		footer = m.searchMsg + " (Press any key)"
	} else {
		footer = "Nav: Arrows (Ln), PgUp/PgDn (Scr), Tab (Mode), / (Scheme), J (Jump), S (Search), Q (Exit)"
	}
	lines = append(lines, footer)

	// Ensure we don't exceed height
	if len(lines) > m.height {
		lines = lines[:m.height]
	}

	return strings.Join(lines, "\n")
}

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

	data, err := syscall.Mmap(int(file.Fd()), 0, int(fileSize), syscall.PROT_READ, syscall.MAP_PRIVATE)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error memory mapping file: %v\n", err)
		os.Exit(1)
	}
	defer syscall.Munmap(data)

	ti := textinput.New()
	ti.Placeholder = "Value..."
	ti.CharLimit = 156
	ti.Width = 40

	m := model{
		data:          data,
		fileSize:      fileSize,
		filename:      filename,
		globalEntropy: calculateEntropy(data),
		textInput:     ti,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}

// Ported helper functions

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

		var magicInfo string
		if rowOffset < fileSize {
			detectLen := 262
			if int64(detectLen) > fileSize-rowOffset {
				detectLen = int(fileSize - rowOffset)
			}
			kind, _ := filetype.Match(data[rowOffset : rowOffset+int64(detectLen)])
			if kind != filetype.Unknown {
				magicInfo = fmt.Sprintf(" | %s (%s)", kind.Extension, kind.MIME.Value)
			}
		}
		fmt.Fprintf(w, "%s\n", magicInfo)
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
		fmt.Fprint(w, "\n")
	}
}

func renderHilbert(w io.Writer, data []byte, fileSize int64, offset int64, n, displayRows int, scheme colorScheme) {
	currentOffset := offset
	rowsRemaining := displayRows

	for rowsRemaining > 0 && currentOffset < fileSize {
		rowsToRender := n
		if rowsToRender > rowsRemaining {
			rowsToRender = rowsRemaining
		}

		grid := make([][]byte, n)
		mask := make([][]bool, n)
		for i := range grid {
			grid[i] = make([]byte, n)
			mask[i] = make([]bool, n)
		}

		for d := 0; d < n*n; d++ {
			var x, y int
			d2xy(n, d, &x, &y)
			addr := currentOffset + int64(d)
			if addr < fileSize {
				grid[y][x] = data[addr]
				mask[y][x] = true
			}
		}

		for y := 0; y < rowsToRender; y++ {
			for x := 0; x < n; x++ {
				if mask[y][x] {
					fmt.Fprintf(w, "%s  \033[0m", getColor(grid[y][x], scheme))
				} else {
					fmt.Fprint(w, "  ")
				}
			}
			fmt.Fprint(w, "\n")
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

func getHeaderLines(scheme colorScheme) []string {
	var lines []string
	if scheme == schemeRanges {
		lines = append(lines, "Color-coded byte viewer (Ranges):")
		lines = append(lines, " 00-0F: \033[41m  \033[0m 10-1F: \033[42m  \033[0m 20-2F: \033[43m  \033[0m 30-3F: \033[44m  \033[0m")
		lines = append(lines, " 40-4F: \033[45m  \033[0m 50-5F: \033[46m  \033[0m 60-6F: \033[47m  \033[0m 70-7F: \033[1;47m  \033[0m")
		lines = append(lines, " 80-8F: \033[41m  \033[0m 90-9F: \033[42m  \033[0m A0-AF: \033[43m  \033[0m B0-BF: \033[44m  \033[0m")
		lines = append(lines, " C0-CF: \033[45m  \033[0m D0-DF: \033[46m  \033[0m E0-EF: \033[47m  \033[0m F0-FF: \033[1;47m  \033[0m")
		lines = append(lines, "")
	} else if scheme == scheme256Colors {
		lines = append(lines, "Color-coded byte viewer (256-color):")
		lines = append(lines, " Byte values 0-15: \033[48;5;0m  \033[0m \033[48;5;1m  \033[0m \033[48;5;2m  \033[0m \033[48;5;3m  \033[0m")
		lines = append(lines, " Byte values 16-31: \033[48;5;16m  \033[0m \033[48;5;17m  \033[0m \033[48;5;18m  \033[0m \033[48;5;19m  \033[0m")
		lines = append(lines, " Byte values 32-47: \033[48;5;32m  \033[0m \033[48;5;33m  \033[0m \033[48;5;34m  \033[0m \033[48;5;35m  \033[0m")
		lines = append(lines, " Byte values 48-63: \033[48;5;48m  \033[0m \033[48;5;49m  \033[0m \033[48;5;50m  \033[0m \033[48;5;51m  \033[0m")
		lines = append(lines, "")
	} else {
		lines = append(lines, "Color-coded byte viewer (Printable):")
		lines = append(lines, " Null: \033[40m  \033[0m Space: \033[44m  \033[0m Print: \033[42m  \033[0m Other: \033[41m  \033[0m")
		lines = append(lines, "")
		lines = append(lines, "")
		lines = append(lines, "")
		lines = append(lines, "")
	}
	return lines
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

	if scheme == scheme256Colors {
		// Use 256-color terminal codes for better visual distinction
		// We'll use a mapping that colors each byte value with a distinct color
		// from the 256-color palette based on its value
		return fmt.Sprintf("\033[48;5;%dm", value)
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
