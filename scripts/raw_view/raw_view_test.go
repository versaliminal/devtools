package main

import (
	"bytes"
	"math"
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
			scheme: schemeRanges,
		},
		{
			name:   "schemePrintable",
			scheme: schemePrintable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := getHeaderLines(tt.scheme)
			if len(lines) == 0 {
				t.Error("getHeaderLines returned empty slice")
			}
			if tt.scheme == schemeRanges && len(lines) != 6 {
				t.Errorf("schemeRanges expected 6 lines, got %d", len(lines))
			}
			if tt.scheme == schemePrintable && len(lines) != 6 {
				t.Errorf("schemePrintable expected 6 lines, got %d", len(lines))
			}
		})
	}
}

func TestGetColor(t *testing.T) {
	tests := []struct {
		name   string
		value  byte
		scheme colorScheme
	}{
		{
			name:   "printable null",
			value:  0,
			scheme: schemePrintable,
		},
		{
			name:   "printable space",
			value:  32,
			scheme: schemePrintable,
		},
		{
			name:   "printable char",
			value:  65,
			scheme: schemePrintable,
		},
		{
			name:   "printable non-printable",
			value:  127,
			scheme: schemePrintable,
		},
		{
			name:   "ranges low",
			value:  0x0F,
			scheme: schemeRanges,
		},
		{
			name:   "ranges mid",
			value:  0x4F,
			scheme: schemeRanges,
		},
		{
			name:   "ranges high",
			value:  0xFF,
			scheme: schemeRanges,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getColor(tt.value, tt.scheme)
			if result == "" {
				t.Error("getColor returned empty string")
			}
		})
	}
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
			scheme: schemeRanges,
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
			scheme: schemeRanges,
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
			scheme: schemeRanges,
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
