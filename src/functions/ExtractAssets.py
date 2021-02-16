import logging
import os
import re as regex
import ntpath
import UnityPy
# from xml.etree import ElementTree

from classes import Constants
from classes import logger, IndentFilter
from pathlib import Path

from functions.File import *


def extract_unity_assets(input_dir, output_path):

    file_patterns = [
        "^globalgamemanagers",
        "^level[0-9]",
        "^resources",
        "^sharedassets[0-9]"
    ]

    ignored_exts = [
        ".resS",
        ".resource"
    ]

    logger.log(logging.INFO, "Extracting build assets...")
    IndentFilter.level += 1

    # Iterate files
    for file_name in os.listdir(input_dir):
        file_path = os.path.join(input_dir, file_name)

        if not os.path.isfile(file_path):
            continue

        continue_loop = False

        for pattern in file_patterns:
            pattern = regex.compile(pattern)
            if pattern.search(file_name):
                continue_loop = True
                break

        for ext in ignored_exts:
            if file_name.endswith(ext):
                continue_loop = False
                break

        if not continue_loop:
            continue

        extract_assets(file_path, output_path)

    IndentFilter.level -= 1
    logger.log(logging.INFO, "Build assets extracted!")


def extract_assets(file_path, output_path):

    file_name = Path(file_path).name
    logger.log(logging.INFO, f"Extracting assets from \"{file_name}\"")
    IndentFilter.level += 1

    obj_type_len = 0  # 13
    obj_name_len = 0  # 35
    path_id_len = 0  # 6

    env = UnityPy.load(file_path)
    for obj in env.objects:

        obj_types = ["TextAsset", "Sprite", "Texture2D", "AudioClip"]
        if obj.type not in obj_types:
            continue

        data = obj.read()

        obj_name = data.name
        if obj_name == "":
            obj_name = "Untitled"

        if obj.type == "TextAsset":
            first_line = data.text.partition("\n")[0]

            ext = "txt"
            if first_line.startswith("<!DOCTYPE html>"):
                ext = "html"
            elif first_line.startswith("<") or "xml" in first_line:
                ext = "xml"
            elif first_line.startswith("{") or first_line.startswith("["):
                ext = "json"

            output_file = output_path / str(obj.type) / f"{obj_name}.{ext}"
            write_file(output_file, data.script, "wb")

        elif obj.type == "Sprite" or obj.type == "Texture2D":
            # print pathid or something like that here
            output_file = output_path / str(obj.type) / f"{obj_name}.png"
            Path(output_file).parent.mkdir(parents=True, exist_ok=True)

            try:
                data.image.save(output_file)
            except Exception as e:
                logger.log(logging.ERROR, f"Error saving {str(obj.type)} \"{obj_name}\" (Path ID: {obj.path_id} in {file_name}) Error: {e}")

        elif obj.type == "AudioClip":
            for name, data in data.samples.items():
                output_file = output_path / str(obj.type) / name
                write_file(output_file, data, "wb")

        if output_file != "":

            # logger.log(logging.INFO, f"{str(obj.type)} {obj_name} {obj.path_id} {file_name}")
            # logger.log(logging.INFO, f"{str(obj.type):<13} {obj_name:<35} {obj.path_id:<6} {file_name}")

            if len(str(obj.type)) > obj_type_len:
                obj_type_len = len(str(obj.type)) + 1
            if len(obj_name) > obj_name_len:
                obj_name_len = len(obj_name) + 1
            if len(str(obj.path_id)) > path_id_len:
                path_id_len = len(str(obj.path_id)) + 2

            logger.log(logging.INFO, "{:<{}} {:<{}} {:<{}}".format(
                obj_name, obj_name_len,
                str(obj.type), obj_type_len,
                f"(Path ID: {obj.path_id})", path_id_len
            ))

    IndentFilter.level -= 1


def extract_exalt_version(metadata_file: Path, output_file: Path):
    """ Attempts to find the current version string (e.g. `1.3.2.0.0`) located in `global-metadata.dat` """

    # TODO: Decode/decrypt build version from appsettings
    # or we could compare all 1.2.3.4.5 strings in the metadata and use the one closest to the
    # current one (in repo_dir)

    # A simple regex to capture "1.3.2.0.0" isn't as simple as there are many
    # strings that match. It just happens that the version string appears in
    # global-metadata.dat with "127.0.0.1" before it.

    # There are a number (8?) ASCII control characters after 127.0.0.1:
    # 02  08 00  00  09 00  00  00
    # STX BS NUL NUL HT NUL NUL NUL

    # For testing:
    # cat global-metadata.dat | grep --text -Po "127\.0\.0\.1[\x00-\x20]*(\d(?:\.\d){4})"

    logger.log(logging.INFO, "Attempting to extract Exalt version string")
    IndentFilter.level += 1

    pattern = regex.compile(b"127\.0\.0\.1[\x00-\x20]*(\d(?:\.\d){4})")

    with open(metadata_file, "rb") as file:
        data = file.read()
        result = pattern.findall(data)

        if len(result) == 1:
            version_string = result[0].decode("utf-8")
            logger.log(logging.INFO, f"Exalt version is \"{version_string}\"")
            write_file(output_file, version_string)
        else:
            logger.log(logging.INFO, "Could not extract version string! Must be manually updated.")
            write_file(output_file, "")

        IndentFilter.level -= 1


def merge_xml_files(manifest_file: Path, input_dir: Path, output_dir: Path):
    logger.log(logging.INFO, f"Merging xml files...")
    IndentFilter.level += 1

    if not manifest_file.exists():
        logger.log(logging.ERROR, f"Unable to find {manifest_file} !")
        IndentFilter.level -= 1
        return

    manifest = read_json(manifest_file)
    for output_file_name in manifest:
        xml_files = []
        file_names = []
        for merge_file in manifest[output_file_name]:
            if not isinstance(merge_file, dict):
                continue

            merge_file_path = merge_file.get("path")
            if not merge_file_path:
                continue

            file_name = ntpath.basename(merge_file_path)
            if not file_name.endswith("xml"):
                continue

            file_path = input_dir / "TextAsset" / file_name
            if not os.path.isfile(file_path):
                logger.log(logging.ERROR, f"Could not find {file_path} !")
                continue

            xml_files.append(file_path)
            file_names.append(file_name)

        if len(xml_files) == 0:
            continue

        logger.log(logging.DEBUG, f"Merging {len(xml_files)} files. {file_names}")
        merged = merge_xml(xml_files)

        output_file = output_dir / "xml" / f"{output_file_name}.xml"
        write_file(output_file, merged, overwrite=False)
        logger.log(logging.INFO, f"Successfully merged {len(file_names)} files into {output_file_name}.xml")

        # TODO: convert to json (see nrelay code)

    IndentFilter.level -= 1
