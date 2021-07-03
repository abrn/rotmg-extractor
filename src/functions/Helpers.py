import math
import os
import subprocess
import shutil
import json
import logging
import re as regex
from pathlib import Path
from xml.etree import ElementTree

from PIL import Image, ImageFilter

from classes import logger, IndentFilter


def parse_int(str):
    if type(str) is int:
        return str
    
    base = 10
    if "0x" in str:
        base = 0
    return int(str, base)


def strip_non_alphabetic(string: str, replacement=""):
    stripped = regex.sub("[^a-zA-Z]", replacement, string)
    return stripped


def delete_dir_contents(dir_path, hidden_files=False):
    """ use `shutil.rmtree` instead """
    for filename in os.listdir(dir_path):
        if not hidden_files:
            if filename.startswith("."):
                continue

        file_path = os.path.join(dir_path, filename)
        try:
            if os.path.isfile(file_path) or os.path.islink(file_path):
                os.unlink(file_path)
            elif os.path.isdir(file_path):
                shutil.rmtree(file_path)
        except Exception as e:
            logging.error(f"Failed to delete {file_path}. Reason {e}")


def create_dir(path):
    Path(path).mkdir(parents=True, exist_ok=True)


def read_json(file_path):
    with open(file_path) as json_file:
        json_data = json.load(json_file)
        return json_data


def read_file(file_path):
    with open(file_path) as file:
        return file.read()


def find_path(dir: Path, search):
    """ Returns the path to a file or directory by search. Uses glob search. """
    result = list(dir.glob(search))
    if result == []:
        return None

    return next(iter(result))


def rename_duplicate_file(file_path, sep="-"):
    """ Rename a file path if there is a duplicate file. E.g. Untitled-1 or Untitled-2 """

    uniq = 1
    while os.path.isfile(file_path):
        file_name = file_path.stem

        # instead of name-1-2-3 do name-3
        if file_name.endswith(f"{sep}{uniq-1}"):
            file_name = file_name.replace(f"{sep}{uniq-1}", "")

        file_ext = file_path.suffix
        file_path = file_path.parent / f"{file_name}{sep}{uniq}{file_ext}"
        uniq += 1

    return file_path


def write_file(file_path: Path, data, mode="w", overwrite=False, rename_duplicate=True):

    Path(file_path).parent.mkdir(parents=True, exist_ok=True)

    # Save the file as Filename-1.txt if there is a duplicate
    if not overwrite:
        if os.path.isfile(file_path):
            if rename_duplicate:
                file_path = rename_duplicate_file(file_path)
            else:
                logger.log(logging.ERROR, f"Error saving {file_path} ! (overwrite={overwrite}, rename_duplicate={rename_duplicate})")
                return

    with open(file_path, mode) as file:
        file.write(data)


def merge_xml(files):
    xml_data = None
    for file_name in files:
        data = ElementTree.parse(file_name).getroot()
        if xml_data is None:
            xml_data = data
        else:
            xml_data.extend(data)

    return ElementTree.tostring(xml_data).decode("utf-8")


def fix_xml(xml_lines: str) -> str:
    """ Removes non valid XML lines """

    output_lines = []
    in_tag = True
    for line in xml_lines.split("\n"):

        include_line = False

        # Opening Tag: `<Texture>`
        if regex.match(r"^\s*<\w+>$", line):
            in_tag = True
            include_line = True

        # Opening Tag: `<Texture>`
        elif regex.match(r"^\s*<\/\w+>$", line):
            in_tag = False
            include_line = True

        # Any </Tag/>
        elif regex.match(r"^\s*<", line):
            include_line = True

        if in_tag:
            include_line = True

        if include_line:
            output_lines.append(line)

    return "\n".join(output_lines)


def archive_build_files(input_path: Path, output_path: Path, archive: bool, file_name="build_files", format="zip"):

    if archive:
        logger.log(logging.INFO, "Archiving build files...")
        IndentFilter.level += 1

        shutil.make_archive(
            base_name=output_path / file_name,
            format=format,
            root_dir=input_path
        )

        logger.log(logging.INFO, f"Build files archived ({output_path / file_name}.{format})")
        IndentFilter.level -= 1
    else:
        logger.log(logging.INFO, "Copying build files...")
        IndentFilter.level += 1

        shutil.copytree(input_path, output_path / file_name)

        logger.log(logging.INFO, f"Build files copied ({output_path / file_name})")
        IndentFilter.level -= 1
    

def diff_directories(left_dir: Path, right_dir: Path):
    logger.log(logging.INFO, f"Diff directories: {left_dir} {right_dir}")

    diff_proc = subprocess.Popen(["diff", "--recursive", left_dir, right_dir], stdout=subprocess.PIPE)
    lines = [ line.decode("utf-8") for line in diff_proc.stdout.readlines()]

    new_files = sum(1 for line in lines if line.startswith(f"Only in {left_dir}"))
    del_files = sum(1 for line in lines if line.startswith(f"Only in {right_dir}"))

    new_lines = sum(1 for line in lines if line.startswith(f">"))
    del_lines = sum(1 for line in lines if line.startswith(f"<"))

    return (new_files, del_files, new_lines, del_lines)


def remove_transparency(img: Image.Image, bg_colour=(255, 255, 255)):
    alpha = img.getchannel("A")
    new_image = Image.new("RGBA", img.size, bg_colour)
    new_image.paste(img, mask=alpha)
    # return new_image.convert("RGB")
    return new_image


def expand_image(img: Image.Image, amount):
    old_width, old_height = img.size
    canvas_width = old_width + amount * 2
    canvas_height = old_height + amount * 2
    
    new_image = Image.new(img.mode, (canvas_width, canvas_height))
    new_image.paste(img, (amount, amount, amount + old_width, amount + old_height))
    return new_image

def scale_image(img: Image.Image, scale: int, resample=Image.NEAREST):
    if scale == 1:
        return img

    width, height = img.size
    scaled_size = (width * scale, height * scale)
    return img.resize(scaled_size, resample)


def is_pixel_alpha(pixel):
    return pixel[3] != 0


def outline_image(img: Image.Image, outline_width=1):

    # Iterate all pixels in the image
    # if the pixel is transparent and has a coloured pixel next to it, then it can be outlined.
    outlined_pixels = []

    for x in range(img.width):
        for y in range(img.height):
            current_pixel = (x, y)
            color = img.getpixel(current_pixel)

            # ignore if the pixel already has a color
            if color[3] != 0:
                continue

            coords = []
            for i in range(1, outline_width + 1):
                temp_coords = [
                    (x, y),     # center
                    (x+i, y),   # right
                    (x-i, y),   # left
                    (x, y+i),   # up
                    (x, y-i),   # down

                    (x+i, y+i), # top-right
                    (x+i, y-i), # bottom-right
                    (x-i, y+i), # top-left
                    (x-i, y-i), # bottom-left
                ]

                coords += temp_coords

            should_outline = False
            for coord in coords:
                if coord[0] <= -1 or coord[1] <= -1: continue # should probably log that outline is too big for the image, or maybe we just run expand_image?
                if coord[0] >= img.width: continue
                if coord[1] >= img.height: continue
                if coord in outlined_pixels: continue
                if should_outline: break

                color = img.getpixel(coord)
                if color[3] != 0:
                    should_outline = True
                
            if should_outline:
                # img.putpixel(current_pixel, (0, 0, 0, 255))
                outlined_pixels.append(current_pixel)

    return img