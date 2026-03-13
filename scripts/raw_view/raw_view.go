package main

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/h2non/filetype"
)

const (
	hexdumpBytesPerRow = 16

	lipBgColor        = lipgloss.Color("#363239")
	lipTextColor      = lipgloss.Color("#DDDDDD")
	lipHighlightColor = lipgloss.Color("#cfae23")
	lipBorderColor    = lipgloss.Color("#a38ba3")
	lipDimTextColor   = lipgloss.Color("#888888")
	lipBlackColor     = lipgloss.Color("#000000")
	lipBlueColor      = lipgloss.Color("4")
	lipGreenColor     = lipgloss.Color("2")
	lipYellowColor    = lipgloss.Color("3")
	lipRedColor       = lipgloss.Color("1")
)

var (
	dimStyle = lipgloss.NewStyle().
			Foreground(lipDimTextColor).
			Background(lipBgColor)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipTextColor).
			Background(lipBgColor)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("7")).
			Background(lipBgColor).
			Padding(0, 1)

	errorStatusStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("9")).
				Bold(true).
				Background(lipBgColor)

	inputStyle = lipgloss.NewStyle().
			Foreground(lipHighlightColor).
			Background(lipBorderColor)

	highlightTextStyle = lipgloss.NewStyle().
				Foreground(lipHighlightColor).
				Background(lipBgColor).
				Bold(true)

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipBorderColor).
			BorderBackground(lipBgColor)

	blackByteStyle = lipgloss.NewStyle().
			Background(lipBlackColor).
			Foreground(lipTextColor)

	blueByteStyle = lipgloss.NewStyle().
			Background(lipBlueColor).
			Foreground(lipBlackColor)

	greenByteStyle = lipgloss.NewStyle().
			Background(lipGreenColor).
			Foreground(lipBlackColor)

	redByteStyle = lipgloss.NewStyle().
			Background(lipRedColor).
			Foreground(lipBlackColor)
)

type viewMode int

const (
	modeHexdump viewMode = iota
	modeLinear
	modeHilbert
	countOfViewModes
)

type colorScheme int

const (
	scheme8colors colorScheme = iota
	scheme256Colors
	schemePrintable
	countOfColorSchemes
)

var std8Colors = [...]lipgloss.Style{
	lipgloss.NewStyle().Background(lipgloss.Color("0")).Foreground(lipTextColor),   // Black
	lipgloss.NewStyle().Background(lipgloss.Color("4")).Foreground(lipBlackColor),  // Blue
	lipgloss.NewStyle().Background(lipgloss.Color("6")).Foreground(lipBlackColor),  // Cyan
	lipgloss.NewStyle().Background(lipgloss.Color("2")).Foreground(lipBlackColor),  // Green
	lipgloss.NewStyle().Background(lipgloss.Color("3")).Foreground(lipBlackColor),  // Yellow
	lipgloss.NewStyle().Background(lipgloss.Color("1")).Foreground(lipBlackColor),  // Red
	lipgloss.NewStyle().Background(lipgloss.Color("5")).Foreground(lipBlackColor),  // Magenta
	lipgloss.NewStyle().Background(lipgloss.Color("15")).Foreground(lipBlackColor), // White
}

var colorGradient = [256]lipgloss.Style{}

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
	var cmd tea.Cmd = nil
	var model model = m

	if m.searching || m.jumping {
		model, cmd = m.handleSearchInput(msg)
	} else {
		model, cmd = m.handleMessage(msg)
	}

	if model.offset < 0 || model.fileSize == 0 {
		model.offset = 0
	} else if model.offset >= model.fileSize {
		model.offset = model.fileSize - 1
	}

	return model, cmd
}

func (m model) handleSearchInput(msg tea.Msg) (model, tea.Cmd) {
	var cmd tea.Cmd
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
						m.searchMsg = highlightTextStyle.Render(fmt.Sprintf("Found at offset: %08x (Enter: go | ESC: cancel)", idx))
					} else {
						m.searchMsg = errorStatusStyle.Render("String not found.")
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
						m.searchMsg = highlightTextStyle.Render(fmt.Sprintf("Jump to offset: %08x (Enter: go | ESC: cancel)", newOffset))
					} else {
						m.searchMsg = errorStatusStyle.Render("Invalid hex offset.")
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

func (m model) handleMessage(msg tea.Msg) (model, tea.Cmd) {
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
			m.currentMode = (m.currentMode + 1) % countOfViewModes
		case "/":
			m.currentScheme = (m.currentScheme + 1) % countOfColorSchemes

		case "s":
			m.searching = true
			m.jumping = false
			m.textInput.Focus()
			m.textInput.SetValue("")
			m.textInput.Prompt = highlightTextStyle.Render("Search ASCII: ")
			m.searchMsg = inputStyle.Render("")
			m.pendingOffset = -1
			return m, nil

		case "j":
			m.jumping = true
			m.searching = false
			m.textInput.Focus()
			m.textInput.SetValue("")
			m.textInput.Prompt = highlightTextStyle.Render("Jump to Hex Offset (e.g. 1A0): ")
			m.searchMsg = inputStyle.Render("")
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
	return m, nil
}

func (m model) getDisplayRows() int {
	fixedHeight := 3 + 2 + 1 // Borders + Header + Footer
	displayRows := m.height - fixedHeight
	if displayRows < 1 {
		displayRows = 1
	}
	return displayRows
}

func (m model) getStep() int64 {
	hilbertN := m.getHilbertN()
	switch m.currentMode {
	case modeHexdump:
		return int64(hexdumpBytesPerRow)
	case modeLinear:
		return int64(hilbertN * 2)
	case modeHilbert:
		return int64(hilbertN)
	default:
		return 1
	}
}

func (m model) getPageStep() int64 {
	displayRows := m.getDisplayRows()
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
	displayRows := m.getDisplayRows()
	hilbertN := m.getHilbertN()

	headerContent := m.renderHeader(displayRows, hilbertN)
	dataContent := m.renderData(displayRows, hilbertN)
	footerContent := m.renderFooter()

	maxWidth := m.calculateMaxWidth(headerContent, dataContent, footerContent, hilbertN)

	headerView := borderStyle.BorderBottom(false).Width(maxWidth).Background(lipBgColor).Align(lipgloss.Center).Render(headerContent)
	contentView := borderStyle.Width(maxWidth).Background(lipBgColor).Render(dataContent)
	footerView := lipgloss.NewStyle().Width(maxWidth + 2).Background(lipBgColor).Align(lipgloss.Center).Render(footerContent)

	fullView := lipgloss.JoinVertical(lipgloss.Center, headerView, contentView, footerView)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, fullView)
}

func (m model) renderHeader(displayRows int, hilbertN int) string {
	var mappingLine strings.Builder
	mappingLine.WriteString(infoStyle.Render("Color Scheme: "))
	mappingLine.WriteString(getColorMappingHeader(m.currentScheme))
	mappingLine.WriteString(infoStyle.Render(" "))

	viewEnd := m.calculateViewEnd(displayRows, hilbertN)
	viewEntropy := calculateEntropy(m.data[m.offset:viewEnd])

	var statusLine strings.Builder
	statusLine.WriteString(infoStyle.Render("File: "))
	statusLine.WriteString(highlightTextStyle.Render(filepath.Base(m.filename)))
	statusLine.WriteString(infoStyle.Render("   Mode: "))
	statusLine.WriteString(highlightTextStyle.Render(m.getModeName()))
	statusLine.WriteString(infoStyle.Render("   Offset: "))
	statusLine.WriteString(highlightTextStyle.Render(fmt.Sprintf("%08x", m.offset)))
	statusLine.WriteString(infoStyle.Render("/"))
	statusLine.WriteString(highlightTextStyle.Render(fmt.Sprintf("%08x", m.fileSize)))
	statusLine.WriteString(infoStyle.Render("   Entropy: "))
	statusLine.WriteString(highlightTextStyle.Render(fmt.Sprintf("%.3f", viewEntropy)))
	statusLine.WriteString(infoStyle.Render("/"))
	statusLine.WriteString(highlightTextStyle.Render(fmt.Sprintf("%.3f", m.globalEntropy)))
	statusLine.WriteString(infoStyle.Render(""))

	return mappingLine.String() + "\n" + statusLine.String()
}

func (m model) calculateViewEnd(displayRows int, hilbertN int) int64 {
	var visibleBytes int
	switch m.currentMode {
	case modeHexdump:
		visibleBytes = displayRows * hexdumpBytesPerRow
	case modeLinear:
		visibleBytes = displayRows * (hilbertN * 2)
	case modeHilbert:
		visibleBytes = (displayRows / hilbertN) * (hilbertN * hilbertN)
	}
	viewEnd := m.offset + int64(visibleBytes)
	if viewEnd > m.fileSize {
		return m.fileSize
	}
	return viewEnd
}

func (m model) getModeName() string {
	switch m.currentMode {
	case modeHexdump:
		return "Hexdump"
	case modeLinear:
		return "Linear "
	case modeHilbert:
		return "Hilbert"
	default:
		return "Unknown"
	}
}

func (m model) renderData(displayRows int, hilbertN int) string {
	var dataBuf strings.Builder
	switch m.currentMode {
	case modeHexdump:
		renderHexdump(&dataBuf, m.data, m.fileSize, m.offset, displayRows, m.currentScheme)
	case modeLinear:
		renderLinear(&dataBuf, m.data, m.fileSize, m.offset, hilbertN*2, displayRows, m.currentScheme)
	case modeHilbert:
		renderHilbert(&dataBuf, m.data, m.fileSize, m.offset, hilbertN, displayRows, m.currentScheme)
	}
	return strings.TrimSuffix(dataBuf.String(), "\n")
}

func (m model) renderFooter() string {
	if m.searching || m.jumping {
		return m.textInput.View()
	}
	if m.searchMsg != "" {
		return inputStyle.Render(m.searchMsg)
	}
	return helpStyle.Render("Arrows: Scroll | PgUp/Dn: Page | Tab: Mode | /: Scheme | J: Jump | S: Search | Q: Quit")
}

func (m model) calculateMaxWidth(header, data, footer string, hilbertN int) int {
	widths := []int{
		lipgloss.Width(header),
		lipgloss.Width(data),
		lipgloss.Width(footer),
		117,          // Fixed Hexdump width
		hilbertN * 2, // Visual mode width
	}
	max := 0
	for _, w := range widths {
		if w > max {
			max = w
		}
	}
	return max
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
	buf := bytes.NewBuffer(nil)
	for i := 0; i < rows && (offset+int64(i*hexdumpBytesPerRow)) < fileSize; i++ {
		rowOffset := offset + int64(i*hexdumpBytesPerRow)
		offsetStr := fmt.Sprintf("%x", rowOffset)
		offsetPad := "00000000"
		fmt.Fprintf(buf, infoStyle.Render("%s%s%s"), dimStyle.Render(offsetPad[:8-len(offsetStr)]), infoStyle.Render(offsetStr), dimStyle.Render(": "))
		for j := 0; j < hexdumpBytesPerRow; j++ {
			addr := rowOffset + int64(j)
			if addr < fileSize {
				val := data[addr]
				style := getStyle(val, scheme)
				fmt.Fprintf(buf, "%s%s", style.Render(fmt.Sprintf("%02x", val)), dimStyle.Render(" "))
			} else {
				fmt.Fprint(buf, "   ")
			}
		}
		fmt.Fprint(buf, dimStyle.Render(" | "))
		for j := 0; j < hexdumpBytesPerRow; j++ {
			addr := rowOffset + int64(j)
			if addr < fileSize {
				val := data[addr]
				if val >= 32 && val <= 126 {
					fmt.Fprintf(buf, infoStyle.Render("%c"), val)
				} else {
					fmt.Fprint(buf, dimStyle.Render("."))
				}
			} else {
				fmt.Fprint(buf, " ")
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
				magicInfo = fmt.Sprintf("%s (%s)", kind.Extension, kind.MIME.Value)
			}
		}
		// Pad row to a consistent width to prevent centering jitter
		// Base hexdump width: 8 (offset) + 2 (": ") + 16*3 (hex) + 3 (" | ") + 16 (ascii) = 8+2+48+3+16 = 77
		// We'll use a fixed width for magic info area if we want stability
		fmt.Fprintf(w, "%s%s%s\n", buf.String(), dimStyle.Render(" | "), highlightTextStyle.Render(magicInfo))
		buf.Reset()
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

func getColorMappingHeader(scheme colorScheme) string {
	var mapping strings.Builder
	switch scheme {
	case scheme8colors:
		for i := 0; i < len(std8Colors); i++ {
			style := std8Colors[i]
			mapping.WriteString(style.Render(fmt.Sprintf(" %02x ", i*32)))
		}
	case scheme256Colors:
		for i := 0; i < 256; i += 8 {
			style := getStyle(byte(i), scheme256Colors)
			mapping.WriteString(style.Render(" "))
		}
	case schemePrintable:
		mapping.WriteString(blackByteStyle.Render(" NULL "))
		mapping.WriteString(blueByteStyle.Render(" SPACE "))
		mapping.WriteString(greenByteStyle.Render(" PRINT "))
		mapping.WriteString(redByteStyle.Render(" OTHER "))
	}
	return mapping.String()
}

func getStyle(value byte, scheme colorScheme) lipgloss.Style {
	switch scheme {
	case schemePrintable:
		return getPrintableStyle(value)
	case scheme256Colors:
		return colorGradient[value]
	case scheme8colors:
		fallthrough
	default:
		return get8ColorStyle(value)
	}
}

func getPrintableStyle(value byte) lipgloss.Style {
	switch {
	case value == 0:
		return blackByteStyle
	case value == 32:
		return blueByteStyle
	case value >= 33 && value <= 126:
		return greenByteStyle
	default:
		return redByteStyle
	}
}

func populateGradientColorStyle() {
	for i := 0; i < 256; i++ {
		var r, g, b int
		v := i
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
		colorGradient[i] = lipgloss.NewStyle().Foreground(lipBlackColor).
			Background(lipgloss.Color(fmt.Sprintf("#%02X%02X%02X", r, g, b)))
	}
}

func get8ColorStyle(value byte) lipgloss.Style {
	colorIndex := int(value / 32)
	if colorIndex >= len(std8Colors) {
		colorIndex = len(std8Colors) - 1
	}
	return std8Colors[colorIndex]
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <filename>\n", os.Args[0])
		os.Exit(1)
	}

	filename := os.Args[1]

	populateGradientColorStyle()
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
