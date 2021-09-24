import logging
import os
import json
import subprocess
import re as regex
import ntpath
import UnityPy
from platform import system
from pathlib import Path
# from xml.etree import ElementTree

from classes import Constants
from classes import logger, IndentFilter
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

    sys = os_check()
    logger.log(f"Detected operating system: {sys}!")

    logger.log("Extracting build assets...")
    IndentFilter.level += 1

    # Get the _Data directory (where the unity files are located)
    data_dir = find_path(input_dir, "*_Data")

    # Iterate files
    for file_name in os.listdir(data_dir):
        file_path = os.path.join(data_dir, file_name)

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
    logger.log("Build assets extracted!")


def extract_assets(file_path, output_path):

    file_name = Path(file_path).name
    IndentFilter.level += 1

    obj_type_len = 0  # 13
    obj_name_len = 0  # 35
    path_id_len = 0  # 6
    # load the file path into UnityPy
    env = UnityPy.load(file_path)

    # track the total asset count and type
    obj_count = len(env.objects)
    asset_count = {'total': obj_count}
    logger.log(f"Found {obj_count} total Unity assets")
    
    for obj in env.objects:
        obj_types = ["TextAsset", "Sprite", "Texture2D", "AudioClip", "MonoScript"]
        if obj.type not in obj_types:
            asset_count -= 1
            continue
        else:
            if asset_count.get(obj.type) is None:
                asset_count[obj.type] = 0
            asset_count[obj.type] += 1

        data = obj.read()
        output_file = ""
        output_count = obj_count - asset_count
        logger.log(f"#{output_count}/{obj_count} extracting from {obj.type} '{file_name}'")

        # if the file has no name generate one based on the type
        obj_name = data.name
        if obj_name == "":
            logger.log(f"Found untitled asset #{output_count} - {obj.type}", logging.DEBUG)
            obj_name = f"{obj.type}_untitled_{output_count}"
            ext = "bin"
        
        # parse depending on the asset resource type
        if obj.type == "TextAsset":
            first_line = data.text.partition("\n")[0]
            ext = "txt"
            if first_line.startswith("<!DOCTYPE html>"):
                ext = "html"
            elif first_line.startswith("<") or "xml" in first_line:
                #TODO: extract to flash asset folder
                ext = "xml"
            elif first_line.startswith("{") or first_line.startswith("["):
                #TODO: check type and extract to flash asset folder
                ext = "json"
            else:
                ext = ""

            output_file = output_path / str(obj.type) / f"{obj_name}.{ext}"
            write_file(output_file, data.m_Script, "wb")
            continue

        # save Sprite and Texture2D types to seperate folders
        if obj.type == "Sprite" or obj.type == "Texture2D":
            # print pathid or something like that here
            output_file = output_path / str(obj.type) / f"{obj_name}.png"
            Path(output_file).parent.mkdir(parents=True, exist_ok=True)
            # attempt to save the image to the output path
            try:
                data.image.save(output_file)
            except Exception as e:
                logger.log(logging.ERROR, f"Error saving {str(obj.type)} \"{obj_name}\" (Path ID: {obj.path_id} in {file_name}) Error: {e}")
            finally:
                continue

        # save AudioClip data
        if obj.type == 'AudioClip':
            # TODO: add option to skip audio clips
            for name, data in data.samples.items():
                output_file = os.path.join(output_path, str(obj.type), name)
                write_file(output_file, data, 'wb')

        # extract and save MonoScript assets
        if obj.type == 'MonoScript':
            dirs = data.m_Namespace.split('.')
            # remove invalid file characters
            dirs = [regex.sub('[*?:"<>|]', "", dir) for dir in dirs]
            dir = "/".join(dirs)

            output_file = os.path.join(output_path, str(obj.type), dir, f"{obj_name}.json")

            keys = ["m_AssemblyName", "m_Namespace", "m_ClassName", "name"]
            base = { key: data.__dict__[key] for key in keys}

            json_pretty = json.dumps(base, indent=4)
            write_file(output_file, json_pretty, 'w')

        if output_file != '':
            # logger.log(f"{str(obj.type)} {obj_name} {obj.path_id} {file_name}")
            # logger.log(f"{str(obj.type):<13} {obj_name:<35} {obj.path_id:<6} {file_name}")
            if len(str(obj.type)) > obj_type_len:
                obj_type_len = len(str(obj.type)) + 1
            if len(obj_name) > obj_name_len:
                obj_name_len = len(obj_name) + 1
            if len(str(obj.path_id)) > path_id_len:
                path_id_len = len(str(obj.path_id)) + 2

            logger.log("{:<{}} {:<{}} {:<{}}".format(
                obj_name, obj_name_len,
                str(obj.type), obj_type_len,
                f"(Path ID: {obj.path_id})", path_id_len
            ))

    IndentFilter.level -= 1


def extract_exalt_version(metadata_file: Path, output_file: Path):
    """ Attempts to find the current version string (e.g. `1.3.2.0.0`) located in `global-metadata.dat` """

    # TODO: Decode/decrypt build version from appsettings

    # A simple regex to capture "1.3.2.0.0" isn't as simple as there are many
    # strings that match. However, the current exalt version is stored in the
    # client as a const string (so it appears in the metadata). It's located
    # in the static class KFFELHLKACG.AFOGMBOANMH.
    # Because it is stored in the metadata, we can use regex to match the
    # string using the previous const strings in the class to get the correct
    # one. (Which is 127.0.0.1 - see the class KFFELHLKACG)

    # For testing:
    # cat global-metadata.dat | grep --text -Po "127\.0\.0\.1[\x00-\x20]*(\d(?:\.\d){4})"

    logger.log("Attempting to extract Exalt version string...")
    IndentFilter.level += 1

    pattern = regex.compile(b"127\.0\.0\.1[\x00-\x20]*(\d(?:\.\d){4})")

    version_string = ''
    with open(metadata_file, 'rb') as file:
        data = file.read()
        result = pattern.findall(data)

        if len(result) == 1:
            version_string = result[0].decode('utf-8')
            logger.log(f"Exalt version is '{version_string}'")
            write_file(output_file, version_string)
        else:
            logger.log("Could not extract version string! Must be manually updated", logging.ERROR)
            write_file(output_file, "")

    IndentFilter.level -= 1
    return version_string


def merge_xml_files(manifest_file: Path, input_dir: Path, output_dir: Path):
    logger.log(f"Merging xml files...")
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
            merge_file_path = merge_file.get('path')
            if not merge_file_path:
                continue
            file_name = ntpath.basename(merge_file_path)
            if not file_name.endswith('xml'):
                continue
            file_path = input_dir / "TextAsset" / file_name
            if not os.path.isfile(file_path):
                logger.log(logging.ERROR, f"Could not find {file_path} !")
                continue

            xml_files.append(file_path)
            file_names.append(file_name)
        if len(xml_files) == 0:
            continue
        logger.log(logging.DEBUG, f"Merging {len(xml_files)} filesL {file_names}")
        merged = merge_xml(xml_files)

        output_file = os.path.join(output_dir, 'xml', f"{output_file_name}.xml")
        write_file(output_file, merged, overwrite=False)
        logger.log(f"Successfully merged {len(file_names)} files into {output_file_name}.xml")

        # TODO: convert to json (see nrelay code)
    IndentFilter.level -= 1


def unpack_launcher_assets(launcher_path, output_path):
    sys = os_check()
    unpacker_file = None
    if sys == 'Windows':
        unpacker_file = Constants.LAUNCHER_UNPACKER_WIN
    elif sys == 'Linux':
        unpacker_file = Constants.LAUNCHER_UNPACKER_UNIX
    elif sys == "Darwin":
        unpacker_file = Constants.LAUNCHER_UNPACKER_MAC
    
    logger.log("Unpacking launcher assets...")
    IndentFilter.level += 1

    process = subprocess.Popen(
        [unpacker_file, launcher_path, output_path],
        stdin=subprocess.PIPE, # bypass "Press any key to exit..."
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT
    )

    logger.pipe(process.stdout)
    process.wait()
    logger.log("Done!")
    IndentFilter.level -= 1


def dump_il2cpp(gameassembly: Path, metadata_file: Path, output_dir: Path):
    sys = os_check()
    dumper_file = None
    if sys == 'Windows':
        dumper_file = Constants.IL2CPP_DUMPER_WIN
    elif sys == 'Linux':
        dumper_file = Constants.IL2CPP_DUMPER_UNIX
    elif sys == "Darwin":
        dumper_file = Constants.IL2CPP_DUMPER_MAC

    logger.log("Dumping via Il2CppInspector...")
    IndentFilter.level += 1

    output_dir.mkdir(parents=True, exist_ok=True)

    process = subprocess.Popen(
        [
            dumper_file, 
            "--bin", gameassembly, 
            "--metadata", metadata_file,
            "--layout", "class",
            "--select-outputs",
            "--py-out",   output_dir / "il2cpp.py",
            "--json-out", output_dir / "metadata.json",
            "--cs-out",   output_dir / "types",
            "--cpp-out",  output_dir / "cpp",
        ],
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT
    )

    logger.pipe(process.stdout)
    process.wait()
    logger.log("Done!")
    IndentFilter.level -= 1


def os_check() -> str:
    #TODO move this to a utils file
    os_name = system()
    if os_name not in ["Windows", "Linux", "Darwin"]:
        return None
    return os_name
