import argparse
import random
from pathlib import Path
from PIL import Image, ImageOps, ImageDraw, ImageFont
from termcolor import colored

ASCII_PALETTES = {
    "minimal": '@%#*+=-:. ',
    "smooth": '@#W$9876543210?!abc;:+=-,._',
    "extended": r'$@B%8&WM#*oahkbdpqwmZO0QLCJUYXzcvunxrjft/\|()1{}[]?-_+~<>i!lI;:,"^`\'. ',
    "slices": '█▉▊▋▌▍▎▏ ',
    "bold": '█▓▒░ ',
    "boxes": '■▣▨▢',
    "stairs": '█▇▆▅▄▃▂▁ ',
}
COLOR_PALETTES = {
    "versaliminal": lambda p: versaliminal_palette(p)
}
FONT_LIST = [
    "arialbd.ttf",
    "Arial Bold.ttf",
    "DejaVuSans-Bold.ttf",
    "LiberationSans-Bold.ttf"
]
DEFAULT_WIDTH = 100
CHARACTER_ASPECT_RATIO = 0.55
GREYSCALE_FORMAT= "L"
FONT_PADDING = 15
FONT_IMG_WIDTH = 400
QUIET = False
AVAILABLE_FONT_NAME = ""

def versaliminal_palette(percentage):
    if percentage > 0.6:
        color = random.choice([('magenta', None), ('magenta', ['dark']), ('light_red', None)])
    else:
        color = ('light_magenta', None)
    return color

def get_first_available_font():
    for font_name in FONT_LIST:
        try:
            _ = ImageFont.truetype(font_name, 10)
            return font_name
        except IOError:
            continue
    return ImageFont.load_default()

def find_filling_font(text, font):
    dummy_img = Image.new(GREYSCALE_FORMAT, (1, 1))
    dummy_draw = ImageDraw.Draw(dummy_img)
    best_font_size = 10
    best_font = None
    while best_font_size < 1000:
        current_font_size = best_font_size + 2
        font = ImageFont.truetype(AVAILABLE_FONT_NAME, current_font_size)
        left, top, right, bottom = dummy_draw.textbbox((0, 0), text, font=font)
        text_width = right - left
        text_height = bottom - top

        if text_width + FONT_PADDING > FONT_IMG_WIDTH * 0.95:
            break
        
        best_font_size = current_font_size
        best_font = font

    return (best_font, text_width, text_height)


def generate_text_image(text):
    font, text_width, text_height = find_filling_font(text, None)

    image_width = int(max(FONT_IMG_WIDTH, text_width + FONT_PADDING))
    image_height = int(text_height + FONT_PADDING)

    img = Image.new('RGB', (image_width, image_height), color = (255, 255, 255))
    d = ImageDraw.Draw(img)

    text_x = (image_width - text_width) / 2
    text_y = (image_height - text_height) / 2
    text_y = text_height * -0.1  # Adjust vertical position slightly upwards for better visual balance
    d.text((text_x, text_y), text, fill=(0, 0, 0), font=font)

    return img

def open_image(input):
    try:
        return Image.open(input)
    except FileNotFoundError:
        print(f"Error: Image file does not exist: {input}")
        return None
    except Exception as e:
        print(f"Error opening image: {e}")
        return None

def image_to_ascii(img, palette, palette_name, invert=False, output=None, output_width=DEFAULT_WIDTH, color='white'):
    img = img.convert(GREYSCALE_FORMAT)
    if invert:
        img = ImageOps.invert(img)

    width, height = img.size
    aspect_ratio = height / width
    output_height = int(output_width * aspect_ratio * CHARACTER_ASPECT_RATIO)

    img = img.resize((output_width, output_height))

    pixels = img.get_flattened_data()

    color_func = COLOR_PALETTES.get(color, None)
    if not QUIET:
        if color_func:
            print(f"Using color function: {color}")
        else:
            print(f"Using color: {color}")

    colors = []
    ascii_art = ""
    for pixel_value in pixels:
        index = pixel_value * (len(palette) - 1) // 255
        if color_func:
            colors.append(color_func(index / len(palette)))
        ascii_art += palette[index]

    formatted_ascii_art = ""
    for i in range(0, len(ascii_art), output_width):
        formatted_ascii_art += ascii_art[i:i+output_width] + "\n"

    if output:
        with open(output, "w") as f:
            f.write(formatted_ascii_art)
        if not QUIET:
            print(f"ASCII art successfully written to {output}")
    else:
        if not QUIET:
            print(f"ASCII art for {input} using {palette} palette ({palette_name}):\n")
        if not color_func and color:
            formatted_ascii_art = colored(formatted_ascii_art, color)
            print(formatted_ascii_art)
        if color_func:
            for i, (color, attrs) in enumerate(colors):
                print(colored(formatted_ascii_art[i], color, attrs=attrs), end="")
            print()

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Convert an image to ASCII art.")
    parser.add_argument("-i", "--input-image", help="Path to the input image file.")
    parser.add_argument("-t", "--text", help="Use text string as input instead of an image.")
    parser.add_argument("-o", "--output-path", help="Path to the output files to (if not provided, prints to console).")
    parser.add_argument("-n", "--name", help="Name of the palette to use (see --list-palettes).")
    parser.add_argument("-p", "--palette", default=None,
                        help="Custom ASCII character palette from darkest to lightest.")
    parser.add_argument("-w", "--width", type=int, default=DEFAULT_WIDTH,
                        help=f"Output width of the ASCII art. Default: {DEFAULT_WIDTH}.")
    parser.add_argument("-x", "--invert", action="store_true",
                        help="Invert the colors of the ASCII art.")
    parser.add_argument("--list-palettes", action="store_true",
                        help="List available palette names and exit.")
    parser.add_argument("-q", "--quiet", action="store_true",
                        help="Suppress non-essential output messages.")
    parser.add_argument("-c", "--color", default='white', help="Colorize the ASCII art output (only works in console).")
    args = parser.parse_args()

    QUIET = args.quiet
    AVAILABLE_FONT_NAME = get_first_available_font()
    
    if args.list_palettes:
        print("Available palettes:")
        for name, palette in ASCII_PALETTES.items():
            print(f" - {name}: {palette}")
        exit(0)

    if args.text:
        input_name = f"\"{args.text}\""
        img = generate_text_image(args.text)
    else:
        input_name = Path(args.input_image).stem
        img = open_image(args.input_image)
        if img is None:
            exit(1)

    palettes = ASCII_PALETTES
    if args.palette:
        palettes = {"custom": args.palette}
    elif args.name:
        if args.name not in ASCII_PALETTES:
            print(f"Error: Palette name '{args.name}' not found. Available palettes: {', '.join(ASCII_PALETTES.keys())}")
            exit(1)
        palettes = {args.name: ASCII_PALETTES[args.name]}

    output_path = Path(args.output_path) if args.output_path else None
    for name, palette in palettes.items():
        output = None
        if output_path:
            output = output_path.joinpath(f"{input_name}_{name}.txt")
        image_to_ascii(img, palette, name, args.invert, output, args.width, args.color)
