import os
import subprocess
import shutil
import json
import logging
import xml.etree.ElementTree as xml
from pathlib import Path
from classes import logger, IndentFilter


def delete_dir_contents(dir_path, hidden_files=False):
    """ use `shutil.rmtree` instead """
    for filename in os.listdir(dir_path):
        if not hidden_files and filename.startswith('.'):
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


def read_json(file_path: str) -> object:
    with open(file_path) as json_file:
        json_data = json.load(json_file)
        return json_data


def read_file(file_path: str) -> object:
    with open(file_path) as file:
        return file.read()


def find_path(dir: Path, search: str) -> Path or None:
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
        if os.path.isfile(file_path) and not rename_duplicate:
            logger.log(logging.ERROR, f"Error saving {file_path} ! (overwrite={overwrite}, rename_duplicate={rename_duplicate})")
            return
        file_path = rename_duplicate_file(file_path)
    with open(file_path, mode) as file:
        file.write(data)


def merge_xml(files):
    xml_data = None
    for file_name in files:
        data = xml.parse(file_name).getroot()
        xml_data = data if xml_data is None else xml_data.extend(data)
    return xml.tostring(xml_data).decode("utf-8")


def archive_build_files(input_path: Path, output_path: Path, archive: bool, file_name="build_files", format="zip"):
    if archive:
        logger.log("Archiving build files...")
        IndentFilter.level += 1

        shutil.make_archive(
            base_name=output_path / file_name,
            format=format,
            root_dir=input_path
        )
        logger.log(f"Build files archived ({output_path / file_name}.{format})")
        IndentFilter.level -= 1
    else:
        logger.log("Copying build files...")
        IndentFilter.level += 1

        shutil.copytree(input_path, output_path / file_name)
        logger.log(f"Build files copied ({output_path / file_name})")
        IndentFilter.level -= 1
    pass


def diff_directories(left_dir: Path, right_dir: Path):
    logger.log(f"Diff directories: {left_dir} {right_dir}")

    diff_proc = subprocess.Popen(["diff", "--recursive", left_dir, right_dir], stdout=subprocess.PIPE)
    lines = [line.decode("utf-8") for line in diff_proc.stdout.readlines()]

    new_files = sum(1 for line in lines if line.startswith(f"Only in {left_dir}"))
    del_files = sum(1 for line in lines if line.startswith(f"Only in {right_dir}"))
    new_lines = sum(1 for line in lines if line.startswith('>'))
    del_lines = sum(1 for line in lines if line.startswith('<'))

    return (new_files, del_files, new_lines, del_lines)