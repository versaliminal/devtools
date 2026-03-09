# Terminal Raw Viewer

A terminal-based binary file viewer that displays files in different visualization modes with color-coded byte representation.

## Features

- **Multiple View Modes**:
  - **Hexdump**: Traditional hex view with 16 bytes per row and embedded file detection via magic
  - **Linear**: Wrapped linear byte grid
  - **Hilbert Curve**: Space-filling curve visualization
   - Creates a continuous visualization that maintains locality

- **Color Schemes**:
  - **Ranges**: Color-code bytes by range (00-0F, 10-1F, etc.)
  - **Printable**: Color-code bytes as null, space, printable, or non-printable
  - **256 Colors**: Each byte value mapped to a distinct color in the 256-color palette

- **Interactive Navigation**:
  - Scroll by line (arrow keys)
  - Scroll by page (page up/page down)
  - Jump to specific offset (J key)
  - Search for text strings (S key)
  - Switch between visualization modes (Tab key)
  - Change color schemes (/ key)
  - Exit application (Q or Escape key)

- **Additional Features**:
  - File entropy calculation
  - Embedded file detection via magic numbers
  - Memory-mapped file access for efficient reading

## Running

```bash
go run . <file_to_view>
```

## Usage

### Key Bindings

- **Arrow Keys**: Scroll by line
- **Page Up/Page Down**: Scroll by page
- **Tab**: Switch between view modes (Hexdump, Linear, Hilbert)
- **/**: Change color scheme
- **J**: Jump to specific offset (enter hex offset like 1A0)
- **S**: Search for text string
- **Q** or **Escape**: Quit application