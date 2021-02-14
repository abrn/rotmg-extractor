import logging
import os
import re as regex
import UnityPy
# from xml.etree import ElementTree

from classes import Constants
from classes import logger, IndentFilter
from pathlib import Path

from functions.File import write_file


def extract_all_assets(file_patterns, ignored_exts=[], dir=Constants.FILES_DIR):

    logger.log(logging.INFO, "Extracting build assets...")
    IndentFilter.level += 1

    # Iterate files
    for file_name in os.listdir(dir):
        file_path = os.path.join(dir, file_name)

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

        extract_assets(file_path)

    IndentFilter.level -= 1
    logger.log(logging.INFO, "Build assets extracted!")


def extract_assets(file_path):

    file_name = Path(file_path).name
    logger.log(logging.INFO, f"Extracting assets from \"{file_name}\"")
    IndentFilter.level += 1

    obj_type_len = 0    #13
    obj_name_len = 0    #35
    path_id_len = 0    #6

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
            
            output_file = Constants.WORK_DIR / str(obj.type) / f"{obj_name}.{ext}"
            write_file(output_file, data.script, "wb")

        elif obj.type == "Sprite" or obj.type == "Texture2D":
            # print pathid or something like that here
            output_file = Constants.WORK_DIR / str(obj.type) / f"{obj_name}.png"
            Path(output_file).parent.mkdir(parents=True, exist_ok=True)

            try:
                data.image.save(output_file)
            except Exception as e:
                logging.error(f"Error saving {str(obj.type)} \"{obj_name}\" (Path ID: {obj.path_id} in {file_name}) Error: {e}")

        elif obj.type == "AudioClip":
            for name, data in data.samples.items():
                output_file = Constants.WORK_DIR / str(obj.type) / name
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
                
        


def get_version_string(metadata_file):
    """ Attempts to find the current version string (e.g. `1.3.2.0.0`) located in `global-metadata.dat` """

    # TODO: Decode/decrypt build version from appsettings

    # A simple regex to capture "1.3.2.0.0" isn't as simple as there are many
    # strings that match. It just happens that the version string appears in
    # global-metadata.dat with "127.0.0.1" before it.

    # There are a number (8?) ASCII control characters after 127.0.0.1:
    # 02  08 00  00  09 00  00  00
    # STX BS NUL NUL HT NUL NUL NUL

    # For testing:
    # cat global-metadata.dat | grep --text -Po "127\.0\.0\.1[\x00-\x20]*(\d(?:\.\d){4})"

    pattern = regex.compile(b"127\.0\.0\.1[\x00-\x20]*(\d(?:\.\d){4})")

    with open(metadata_file, "rb") as file:
        data = file.read()
        result = pattern.findall(data)

        if len(result) == 1:
            return result[0].decode("utf-8")

    return None


# def do_manifest_stuff():
#     logger.log(logging.INFO, f"Merging xml files...")
#     rel_dir = relative_dir("./output/current/json/manifest.json")
#     manifest = read_json(rel_dir)

#     # ignore this key as it has no paths
#     del manifest["settings"]

#     for merge_file in manifest:
#         xml_files = []
#         file_names = []
#         for file in manifest[merge_file]:
#             path = file["path"]
#             if not path.endswith("xml"):
#                 continue

#             file_name = ntpath.basename(path)
#             rel_file = relative_dir(f"./output/current/xml/{file_name}")

#             xml_files.append(rel_file)
#             file_names.append(file_name)
#             # data = read_file(rel_file)
#             # xml = xmltodict.parse(data)

#             # Get the first xml key
#             # e.g. GroundTypes Objects EquipmentSets
#             # key = next(iter(xml)) 

#         if xml_files == []:
#             continue

#         logger.log(logging.DEBUG, f"Merging {file_names}")
#         merged = merge_xml(xml_files)

#         save_text(f"output/current/merged", f"{merge_file}.xml", merged)
#         logger.log(logging.INFO, f"Successfully merged {len(file_names)} xml files into {merge_file}.xml")

