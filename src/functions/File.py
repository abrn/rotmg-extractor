import os
import shutil
import json
import logging
from pathlib import Path
from xml.etree import ElementTree

from classes import logger
from classes import Constants

# def delete_dir_contents(dir_path):
#     for filename in os.listdir(dir_path):
#         file_path = os.path.join(dir_path, filename)
#         try:
#             if os.path.isfile(file_path) or os.path.islink(file_path):
#                 os.unlink(file_path)
#             elif os.path.isdir(file_path):
#                 delete_dir_contents(file_path)
#                 shutil.rmtree(file_path)
#         except Exception as e:
#             logging.error(f"Failed to delete {file_path}. Reason {e}")


def create_dir(path):
    Path(path).mkdir(parents=True, exist_ok=True)


def read_json(file_path):
    with open(file_path) as json_file:
        json_data = json.load(json_file)
        return json_data


def read_file(file_path):
    with open(file_path) as file:
        return file.read()


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


def archive_build_assets(input_path: Path, output_path: Path, file_name="build", format="zip"):
    logger.log(logging.INFO, "Archiving build assets...")

    # gztar includes the entire directory structure (C:\Users\...)
    # tar includes "." and @PaxHeaders
    # zip is the only one that actually works
    shutil.make_archive(
        base_name=output_path / file_name,
        format="zip",
        root_dir=input_path
    )

    # rel_output_path = input_path.relative_to(Constants.SRC_DIR)
    logger.log(logging.INFO, f"Build files archived ({output_path / file_name}.{format})")
