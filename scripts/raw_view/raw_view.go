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
	"github.com/charmbracelet/lipgloss"
	"github.com/h2non/filetype"
)

const (
	bytesPerRow = 16
)

// Define styling for better TUI appearance using lipgloss
var (
	// Base styles
	baseStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("235"))

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true).
			Padding(0, 1)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Italic(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("7")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Bold(true)

	errorStatusStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("9")).
				Bold(true)

	// Text input styles
	inputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("234"))

	inputPromptStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("86"))

	lilacStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#C8A2C8"))

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#C8A2C8"))

	// Byte rendering colors (256-color compatible)
	byteColors = []lipgloss.Color{
		lipgloss.Color("196"), // 0x00-0x0F red
		lipgloss.Color("82"),  // 0x10-0x1F green
		lipgloss.Color("226"), // 0x20-0x2F yellow
		lipgloss.Color("21"),  // 0x30-0x3F blue
		lipgloss.Color("201"), // 0x40-0x4F magenta
		lipgloss.Color("51"),  // 0x50-0x5F cyan
		lipgloss.Color("244"), // 0x60-0x6F gray
		lipgloss.Color("255"), // 0x70-0x7F white
		lipgloss.Color("196"), // 0x80-0x8F red
		lipgloss.Color("82"),  // 0x90-0x9F green
		lipgloss.Color("226"), // 0xA0-0xAF yellow
		lipgloss.Color("21"),  // 0xB0-0xBF blue
		lipgloss.Color("201"), // 0xC0-0xCF magenta
		lipgloss.Color("51"),  // 0xD0-0xDF cyan
		lipgloss.Color("244"), // 0xE0-0xEF gray
		lipgloss.Color("255"), // 0xF0-0xFF white
	}

	printableByteColors = map[string]lipgloss.Color{
		"null":      lipgloss.Color("0"),   // Black for null
		"space":     lipgloss.Color("21"),  // Blue for space
		"printable": lipgloss.Color("82"),  // Green for printable
		"other":     lipgloss.Color("196"), // Red for non-printable
	}
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

	// Handle search/jump mode separately
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
				m.searchMsg = ""
				m.pendingOffset = -1
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
		// If we are showing a message, clear it on any key press
		if m.searchMsg != "" {
			if msg.String() == "enter" && m.pendingOffset != -1 {
				m.offset = m.pendingOffset
			}
			m.searchMsg = ""
			m.pendingOffset = -1
		}

		switch msg.String() {
		case "q", "ctrl+c":
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

		case "s":
			m.searching = true
			m.jumping = false
			m.textInput.Focus()
			m.textInput.SetValue("")
			m.textInput.Prompt = "Search ASCII: "
			m.searchMsg = ""
			m.pendingOffset = -1
			return m, nil

		case "j":
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
		return titleStyle.Render("Initializing...") + "\n"
	}

	// Calculate fixed-size component heights
	headerLines := getHeaderLines(m.currentScheme)
	// Header Section: Title (1) + Scheme (depends) + Entropy (1) + Status (1)
	// Plus border (2)
	headerContentHeight := len(headerLines) + 2 // +2 for entropy and status

	// Total fixed height = Header (content + 2 border) + Footer (content + 2 border) + 2 border for content
	fixedHeight := (headerContentHeight + 2) + (1 + 2) + 2

	displayRows := m.height - fixedHeight
	if displayRows < 1 {
		displayRows = 1
	}

	hilbertN := m.getHilbertN()

	// 1. Header Section
	viewEnd := m.offset + int64(displayRows*bytesPerRow) // Approximation for entropy
	if m.currentMode == modeLinear {
		viewEnd = m.offset + int64(displayRows*(m.width/2))
	} else if m.currentMode == modeHilbert {
		viewEnd = m.offset + int64((displayRows/hilbertN)*(hilbertN*hilbertN))
	}
	if viewEnd > m.fileSize {
		viewEnd = m.fileSize
	}
	viewEntropy := calculateEntropy(m.data[m.offset:viewEnd])
	entropyLine := fmt.Sprintf("Entropy: Global: %s bits/byte | View: %s bits/byte",
		lilacStyle.Render(fmt.Sprintf("%.4f", m.globalEntropy)),
		lilacStyle.Render(fmt.Sprintf("%.4f", viewEntropy)))

	modeName := ""
	switch m.currentMode {
	case modeHexdump:
		modeName = "Hexdump"
	case modeLinear:
		modeName = "Wrapped Linear"
	case modeHilbert:
		modeName = "Hilbert Curve"
	}
	statusLine := fmt.Sprintf("File: %s | Mode: %s | Offset: %s / %s",
		m.filename,
		modeName,
		lilacStyle.Render(fmt.Sprintf("%08x", m.offset)),
		lilacStyle.Render(fmt.Sprintf("%08x", m.fileSize)))

	headerCombined := append(headerLines, infoStyle.Render(entropyLine), baseStyle.Render(statusLine))
	headerContent := lipgloss.JoinVertical(lipgloss.Left, headerCombined...)

	// 2. Data rows
	var dataBuf strings.Builder
	switch m.currentMode {
	case modeHexdump:
		renderHexdump(&dataBuf, m.data, m.fileSize, m.offset, displayRows, m.currentScheme)
	case modeLinear:
		renderLinear(&dataBuf, m.data, m.fileSize, m.offset, m.width-4, displayRows, m.currentScheme) // -4 for borders
	case modeHilbert:
		renderHilbert(&dataBuf, m.data, m.fileSize, m.offset, hilbertN, displayRows, m.currentScheme)
	}
	dataLines := strings.Split(strings.TrimSuffix(dataBuf.String(), "\n"), "\n")
	dataContent := lipgloss.JoinVertical(lipgloss.Left, dataLines...)

	// 3. Footer Section
	footer := ""
	if m.searching || m.jumping {
		footer = inputPromptStyle.Render(m.textInput.Prompt) + m.textInput.View()
	} else if m.searchMsg != "" {
		footer = statusStyle.Render(m.searchMsg) + " " + infoStyle.Render("(Enter: go | Esc: cancel)")
	} else {
		footer = helpStyle.Render("Arrows: Scroll | PgUp/Dn: Page | Tab: Mode | /: Scheme | J: Jump | S: Search | Q: Quit")
	}
	footerContent := footer

	// Calculate max width for uniform sections
	w1 := lipgloss.Width(headerContent)
	w2 := lipgloss.Width(dataContent)
	w3 := lipgloss.Width(footerContent)
	maxWidth := w1
	if w2 > maxWidth {
		maxWidth = w2
	}
	if w3 > maxWidth {
		maxWidth = w3
	}

	// Apply uniform width and borders
	headerView := borderStyle.Width(maxWidth).Render(headerContent)
	contentView := borderStyle.Width(maxWidth).Render(dataContent)
	footerView := borderStyle.Width(maxWidth).Render(footerContent)

	// Assemble the whole view
	fullView := lipgloss.JoinVertical(
		lipgloss.Center,
		headerView,
		contentView,
		footerView,
	)

	// Center the entire view within the terminal
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, fullView)
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <filename>\n", os.Args[0])
		os.Exit(1)
	}

	filename := os.Args[1]

	m, err := newModelFromFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing application: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}

// newModelFromFile creates a new model from a file
func newModelFromFile(filename string) (*model, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("getting file stats: %w", err)
	}

	fileSize := stat.Size()
	if fileSize == 0 {
		return nil, fmt.Errorf("file is empty")
	}

	data, err := syscall.Mmap(int(file.Fd()), 0, int(fileSize), syscall.PROT_READ, syscall.MAP_PRIVATE)
	if err != nil {
		return nil, fmt.Errorf("memory mapping file: %w", err)
	}

	ti := textinput.New()
	ti.Placeholder = "Enter value..."
	ti.CharLimit = 156
	ti.Width = 50
	ti.Prompt = "› "

	m := model{
		data:          data,
		fileSize:      fileSize,
		filename:      filename,
		globalEntropy: calculateEntropy(data),
		textInput:     ti,
	}

	return &m, nil
}

// Ported helper functions

func renderHexdump(w io.Writer, data []byte, fileSize int64, offset int64, rows int, scheme colorScheme) {
	for i := 0; i < rows && (offset+int64(i*bytesPerRow)) < fileSize; i++ {
		rowOffset := offset + int64(i*bytesPerRow)
		fmt.Fprintf(w, "%s: ", lilacStyle.Render(fmt.Sprintf("%08x", rowOffset)))
		for j := 0; j < bytesPerRow; j++ {
			addr := rowOffset + int64(j)
			if addr < fileSize {
				val := data[addr]
				style := getStyle(val, scheme)
				fmt.Fprintf(w, "%s ", style.Render(fmt.Sprintf("%02x", val)))
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
		// Pad row to a consistent width to prevent centering jitter
		// Base hexdump width: 8 (offset) + 2 (": ") + 16*3 (hex) + 3 (" | ") + 16 (ascii) = 8+2+48+3+16 = 77
		// We'll use a fixed width for magic info area if we want stability
		fmt.Fprintf(w, "%-40s\n", magicInfo)
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
				style := getStyle(val, scheme)
				fmt.Fprintf(w, "%s", style.Render("  "))
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
					style := getStyle(grid[y][x], scheme)
					fmt.Fprintf(w, "%s", style.Render("  "))
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

	title := titleStyle.Render("Binary File Viewer")
	lines = append(lines, title)

	if scheme == schemeRanges {
		r1 := lipgloss.NewStyle().Background(lipgloss.Color("1")).Render("  ")
		r2 := lipgloss.NewStyle().Background(lipgloss.Color("2")).Render("  ")
		r3 := lipgloss.NewStyle().Background(lipgloss.Color("3")).Render("  ")
		r4 := lipgloss.NewStyle().Background(lipgloss.Color("4")).Render("  ")
		r5 := lipgloss.NewStyle().Background(lipgloss.Color("5")).Render("  ")
		r6 := lipgloss.NewStyle().Background(lipgloss.Color("6")).Render("  ")
		r7 := lipgloss.NewStyle().Background(lipgloss.Color("7")).Render("  ")
		r8 := lipgloss.NewStyle().Background(lipgloss.Color("15")).Render("  ")

		lines = append(lines, "Byte Ranges:")
		lines = append(lines, fmt.Sprintf(" 00: %s 10: %s 20: %s 30: %s 40: %s 50: %s 60: %s 70: %s", r1, r2, r3, r4, r5, r6, r7, r8))
	} else if scheme == scheme256Colors {
		lines = append(lines, "256 Color Gradient (Cool to Warm):")
		var gradient strings.Builder
		for i := 0; i < 256; i += 8 {
			style := getStyle(byte(i), scheme256Colors)
			gradient.WriteString(style.Render(" "))
		}
		lines = append(lines, gradient.String())
	} else {
		nullS := lipgloss.NewStyle().Background(lipgloss.Color("0")).Render("  ")
		spaceS := lipgloss.NewStyle().Background(lipgloss.Color("4")).Render("  ")
		printS := lipgloss.NewStyle().Background(lipgloss.Color("2")).Render("  ")
		otherS := lipgloss.NewStyle().Background(lipgloss.Color("1")).Render("  ")
		lines = append(lines, fmt.Sprintf("Null:%s Space:%s Print:%s Other:%s", nullS, spaceS, printS, otherS))
	}
	return lines
}

func getStyle(value byte, scheme colorScheme) lipgloss.Style {
	if scheme == schemePrintable {
		switch {
		case value == 0:
			return lipgloss.NewStyle().Background(lipgloss.Color("0")) // Null - Black
		case value == 32:
			return lipgloss.NewStyle().Background(lipgloss.Color("4")) // Space - Blue
		case value >= 33 && value <= 126:
			return lipgloss.NewStyle().Background(lipgloss.Color("2")) // Printable - Green
		default:
			return lipgloss.NewStyle().Background(lipgloss.Color("1")) // Non-printable - Red
		}
	}

	if scheme == scheme256Colors {
		// Smooth cool-to-warm gradient
		// 0: Blue (#0000FF), 64: Cyan (#00FFFF), 128: Green (#00FF00), 192: Yellow (#FFFF00), 255: Red (#FF0000)
		var r, g, b int
		v := int(value)
		if v < 64 {
			// Blue to Cyan
			r = 0
			g = v * 4
			b = 255
		} else if v < 128 {
			// Cyan to Green
			r = 0
			g = 255
			b = 255 - (v-64)*4
		} else if v < 192 {
			// Green to Yellow
			r = (v - 128) * 4
			g = 255
			b = 0
		} else {
			// Yellow to Red
			r = 255
			g = 255 - (v-192)*4
			b = 0
		}
		return lipgloss.NewStyle().Background(lipgloss.Color(fmt.Sprintf("#%02X%02X%02X", r, g, b)))
	}

	// Use byteColors slice for ranges scheme
	colorIndex := int(value) / 16
	if colorIndex >= len(byteColors) {
		colorIndex = len(byteColors) - 1
	}

	// Return lipgloss style with the color
	colors := []string{
		"1",  // red
		"2",  // green
		"3",  // yellow
		"4",  // blue
		"5",  // magenta
		"6",  // cyan
		"7",  // light gray
		"15", // white (bright)
		"1",  // red
		"2",  // green
		"3",  // yellow
		"4",  // blue
		"5",  // magenta
		"6",  // cyan
		"7",  // light gray
		"15", // white (bright)
	}

	return lipgloss.NewStyle().Background(lipgloss.Color(colors[colorIndex]))
}
