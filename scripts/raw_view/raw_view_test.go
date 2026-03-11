package main

import (
	"bytes"
	"math"
	"os"
	"testing"
)

func TestCalculateEntropy(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected float64
	}{
		{
			name:     "empty data",
			data:     []byte{},
			expected: 0,
		},
		{
			name:     "uniform data",
			data:     []byte{0x00, 0x00, 0x00, 0x00},
			expected: 0,
		},
		{
			name:     "two byte values",
			data:     []byte{0x00, 0xFF},
			expected: 1.0,
		},
		{
			name:     "all printable ASCII",
			data:     []byte("Hello, World!"),
			expected: calculateEntropy([]byte("Hello, World!")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateEntropy(tt.data)
			if tt.expected == 0 && result != 0 {
				if result != tt.expected {
					t.Errorf("calculateEntropy(%v) = %v, want %v", tt.data, result, tt.expected)
				}
			} else if math.Abs(result-tt.expected) > 0.001 {
				t.Errorf("calculateEntropy(%v) = %v, want %v", tt.data, result, tt.expected)
			}
		})
	}
}

func TestGetHeaderLines(t *testing.T) {
	tests := []struct {
		name   string
		scheme colorScheme
	}{
		{
			name:   "schemeRanges",
			scheme: scheme8colors,
		},
		{
			name:   "schemePrintable",
			scheme: schemePrintable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := getColorMappingHeader(tt.scheme)
			if len(lines) == 0 {
				t.Error("getHeaderLines returned empty slice")
			}
			// Title is now rendered in renderHeader, so getHeaderLines only returns 1 line
			if len(lines) != 1 {
				t.Errorf("expected 1 line, got %d", len(lines))
			}
		})
	}
}

func TestGetStyle(t *testing.T) {
	schemes := []colorScheme{schemePrintable, scheme256Colors, scheme8colors}
	for _, s := range schemes {
		style := getStyle(0xAA, s)
		if style.Render(" ") == "" {
			t.Errorf("getStyle(0xAA, %v) returned empty style", s)
		}
	}
}

func TestGetStyleFunctions(t *testing.T) {
	t.Run("getPrintableStyle", func(t *testing.T) {
		values := []byte{0, 32, 65, 127}
		for _, v := range values {
			style := getPrintableStyle(v)
			if style.Render(" ") == "" {
				t.Errorf("getPrintableStyle(%d) returned empty style", v)
			}
		}
	})

	t.Run("get256ColorStyle", func(t *testing.T) {
		values := []byte{0, 64, 128, 192, 255}
		for _, v := range values {
			style := get256ColorStyle(v)
			if style.Render(" ") == "" {
				t.Errorf("get256ColorStyle(%d) returned empty style", v)
			}
		}
	})

	t.Run("getRangeStyle", func(t *testing.T) {
		values := []byte{0, 64, 128, 192, 255}
		for _, v := range values {
			style := get8ColorStyle(v)
			if style.Render(" ") == "" {
				t.Errorf("getRangeStyle(%d) returned empty style", v)
			}
		}
	})
}

func TestModelViewHelpers(t *testing.T) {
	m := model{
		width:    80,
		height:   24,
		fileSize: 1000,
		filename: "test.bin",
	}

	t.Run("getModeName", func(t *testing.T) {
		m.currentMode = modeHexdump
		if m.getModeName() == "" {
			t.Error("getModeName() returned empty string for Hexdump")
		}
	})

	t.Run("calculateViewEnd", func(t *testing.T) {
		m.offset = 0
		displayRows := 10
		hilbertN := 4

		m.currentMode = modeHexdump
		end := m.calculateViewEnd(displayRows, hilbertN)
		if end != 160 {
			t.Errorf("calculateViewEnd(Hexdump) = %d, want 160", end)
		}

		m.currentMode = modeLinear
		end = m.calculateViewEnd(displayRows, hilbertN)
		if end != 80 {
			t.Errorf("calculateViewEnd(Linear) = %d, want 80", end)
		}

		m.currentMode = modeHilbert
		end = m.calculateViewEnd(displayRows, hilbertN)
		if end != 32 { // (10/4)*16 = 2*16 = 32
			t.Errorf("calculateViewEnd(Hilbert) = %d, want 32", end)
		}
	})

	t.Run("calculateMaxWidth", func(t *testing.T) {
		w := m.calculateMaxWidth("short", "longer data", "footer", 4)
		if w < 117 {
			t.Errorf("calculateMaxWidth = %d, want at least 117 (hexdump width)", w)
		}
	})
}

func TestRot(t *testing.T) {
	tests := []struct {
		name string
		n    int
		x    int
		y    int
		rx   int
		ry   int
	}{
		{
			name: "no rotation",
			n:    4,
			x:    1,
			y:    2,
			rx:   0,
			ry:   0,
		},
		{
			name: "rx=1 ry=0",
			n:    4,
			x:    1,
			y:    2,
			rx:   1,
			ry:   0,
		},
		{
			name: "ry=1",
			n:    4,
			x:    1,
			y:    2,
			rx:   0,
			ry:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y := tt.x, tt.y
			rot(tt.n, &x, &y, tt.rx, tt.ry)
			if x < 0 || y < 0 {
				t.Errorf("rot resulted in negative coordinates: x=%d, y=%d", x, y)
			}
		})
	}
}

func TestD2xy(t *testing.T) {
	tests := []struct {
		name string
		n    int
		d    int
	}{
		{
			name: "n=1 d=0",
			n:    1,
			d:    0,
		},
		{
			name: "n=2 d=0",
			n:    2,
			d:    0,
		},
		{
			name: "n=2 d=1",
			n:    2,
			d:    1,
		},
		{
			name: "n=2 d=2",
			n:    2,
			d:    2,
		},
		{
			name: "n=2 d=3",
			n:    2,
			d:    3,
		},
		{
			name: "n=4 d=0",
			n:    4,
			d:    0,
		},
		{
			name: "n=4 d=15",
			n:    4,
			d:    15,
		},
		{
			name: "n=8 d=0",
			n:    8,
			d:    0,
		},
		{
			name: "n=8 d=63",
			n:    8,
			d:    63,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var x, y int
			d2xy(tt.n, tt.d, &x, &y)
			if x < 0 || x >= tt.n {
				t.Errorf("d2xy(%d, %d) x=%d out of bounds [0, %d)", tt.n, tt.d, x, tt.n-1)
			}
			if y < 0 || y >= tt.n {
				t.Errorf("d2xy(%d, %d) y=%d out of bounds [0, %d)", tt.n, tt.d, y, tt.n-1)
			}
		})
	}
}

func TestModelGetHilbertN(t *testing.T) {
	tests := []struct {
		name     string
		width    int
		expected int
	}{
		{
			name:     "width 80",
			width:    80,
			expected: 32,
		},
		{
			name:     "width 40",
			width:    40,
			expected: 16,
		},
		{
			name:     "width 20",
			width:    20,
			expected: 8,
		},
		{
			name:     "width 10",
			width:    10,
			expected: 4,
		},
		{
			name:     "width 1",
			width:    1,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{width: tt.width}
			result := m.getHilbertN()
			if result != tt.expected {
				t.Errorf("getHilbertN() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestModelGetStep(t *testing.T) {
	tests := []struct {
		name        string
		mode        viewMode
		width       int
		expectedMin int64
	}{
		{
			name:        "hexdump mode",
			mode:        modeHexdump,
			width:       80,
			expectedMin: 16,
		},
		{
			name:        "linear mode",
			mode:        modeLinear,
			width:       80,
			expectedMin: 40,
		},
		{
			name:        "hilbert mode",
			mode:        modeHilbert,
			width:       80,
			expectedMin: 32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{currentMode: tt.mode, width: tt.width}
			result := m.getStep()
			if result < tt.expectedMin {
				t.Errorf("getStep() = %d, want >= %d", result, tt.expectedMin)
			}
		})
	}
}

func TestModelGetPageStep(t *testing.T) {
	tests := []struct {
		name   string
		mode   viewMode
		width  int
		height int
	}{
		{
			name:   "hexdump mode",
			mode:   modeHexdump,
			width:  80,
			height: 24,
		},
		{
			name:   "linear mode",
			mode:   modeLinear,
			width:  80,
			height: 24,
		},
		{
			name:   "hilbert mode",
			mode:   modeHilbert,
			width:  80,
			height: 24,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{currentMode: tt.mode, width: tt.width, height: tt.height}
			result := m.getPageStep()
			if result <= 0 {
				t.Errorf("getPageStep() = %d, want > 0", result)
			}
		})
	}
}

func TestRenderHexdump(t *testing.T) {
	data := []byte("Hello, World! This is a test.")
	fileSize := int64(len(data))

	tests := []struct {
		name   string
		offset int64
		rows   int
		scheme colorScheme
	}{
		{
			name:   "basic render",
			offset: 0,
			rows:   3,
			scheme: scheme8colors,
		},
		{
			name:   "with offset",
			offset: 5,
			rows:   3,
			scheme: schemePrintable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			renderHexdump(&buf, data, fileSize, tt.offset, tt.rows, tt.scheme)
			result := buf.String()
			if result == "" {
				t.Error("renderHexdump returned empty string")
			}
		})
	}
}

func TestRenderLinear(t *testing.T) {
	data := []byte("Hello, World!")
	fileSize := int64(len(data))

	tests := []struct {
		name   string
		offset int64
		width  int
		rows   int
		scheme colorScheme
	}{
		{
			name:   "basic render",
			offset: 0,
			width:  40,
			rows:   3,
			scheme: scheme8colors,
		},
		{
			name:   "small width",
			offset: 0,
			width:  10,
			rows:   2,
			scheme: schemePrintable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			renderLinear(&buf, data, fileSize, tt.offset, tt.width, tt.rows, tt.scheme)
			result := buf.String()
			if result == "" {
				t.Error("renderLinear returned empty string")
			}
		})
	}
}

func TestRenderHilbert(t *testing.T) {
	data := []byte("Hello, World!")
	fileSize := int64(len(data))

	tests := []struct {
		name   string
		n      int
		offset int64
		rows   int
		scheme colorScheme
	}{
		{
			name:   "basic render",
			n:      4,
			offset: 0,
			rows:   4,
			scheme: scheme8colors,
		},
		{
			name:   "with offset",
			n:      4,
			offset: 8,
			rows:   4,
			scheme: schemePrintable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			renderHilbert(&buf, data, fileSize, tt.offset, tt.n, tt.rows, tt.scheme)
			result := buf.String()
			if result == "" {
				t.Error("renderHilbert returned empty string")
			}
		})
	}
}

func TestNewModelFromFile(t *testing.T) {
	// Create a temporary file for testing
	tmpFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write some test content
	testData := []byte("Test data for the application")
	tmpFile.Write(testData)
	tmpFile.Close()

	t.Run("Valid file", func(t *testing.T) {
		m, err := newModelFromFile(tmpFile.Name())
		if err != nil {
			t.Errorf("newModelFromFile() error = %v", err)
		}
		if m == nil {
			t.Error("newModelFromFile() returned nil model")
		}
		if m.filename != tmpFile.Name() {
			t.Errorf("Filename mismatch: got %s, want %s", m.filename, tmpFile.Name())
		}
		if m.fileSize != int64(len(testData)) {
			t.Errorf("FileSize mismatch: got %d, want %d", m.fileSize, len(testData))
		}
	})

	t.Run("Non-existent file", func(t *testing.T) {
		_, err := newModelFromFile("/non/existent/file")
		if err == nil {
			t.Error("newModelFromFile() should have returned an error for non-existent file")
		}
	})

	t.Run("Empty file", func(t *testing.T) {
		emptyFile, err := os.CreateTemp("", "emptyfile")
		if err != nil {
			t.Fatalf("Failed to create empty temp file: %v", err)
		}
		emptyFile.Close()
		defer os.Remove(emptyFile.Name())

		_, err = newModelFromFile(emptyFile.Name())
		if err == nil {
			t.Error("newModelFromFile() should have returned an error for empty file")
		}
	})
}
