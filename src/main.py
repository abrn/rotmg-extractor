import os
import sys
import json
import math
import shutil
import gzip
import urllib.request
import xmltodict
from xml.etree import ElementTree
from pathlib import Path
import ntpath
import logging
from UnityPy import AssetsManager
from datetime import datetime

###################
# File Operations #
###################


def relative_dir(path):
    return Path(__file__).parent.resolve() / path


def create_dir(path):
    Path(path).mkdir(parents=True, exist_ok=True)


def save_text(dir_path, file_name, text):
    rel_path = relative_dir(dir_path)
    create_dir(rel_path)
    output_file = f"{rel_path}/{file_name}"
    with open(output_file, "w") as file:
        # strip non-ascii characters
        text = text.encode("ascii", errors="ignore").decode()

        file.write(text)


def read_file(file_path):
    with open(file_path) as file:
        return file.read()


def read_json(file_path):
    with open(file_path) as json_file:
        json_data = json.load(json_file)
        return json_data


def merge_xml(files):
    xml_data = None
    for file_name in files:
        data = ElementTree.parse(file_name).getroot()
        if xml_data is None:
            xml_data = data
        else:
            xml_data.extend(data)

    return ElementTree.tostring(xml_data).decode("utf-8")


################
# RotMG Builds #
################


def get_build_hash():
    logging.debug("Getting Build Hash")
    url = "https://www.realmofthemadgod.com/app/init?platform=standalonewindows64&key=9KnJFxtTvLu2frXv"
    response = urllib.request.urlopen(url).read()
    data = xmltodict.parse(response)
    build_hash = data["AppSettings"]["BuildHash"]
    logging.info(f"Build Hash is \"{build_hash}\"")
    return build_hash


def download_asset(build_url, url_path, file_name):

    url = build_url + url_path + file_name + ".gz"
    temp_dir = relative_dir("temp")
    output_file = temp_dir / file_name

    # Create directory if not exist
    create_dir(temp_dir)

    # Download file
    logging.info(f"Downloading {url}")
    urllib.request.urlretrieve(url, f"{output_file}.gz")

    # Unzip file
    logging.debug(f"Unzipping {output_file}.gz")
    with gzip.open(f"{output_file}.gz", "rb") as f_in:
        with open(output_file, "wb") as f_out:
            shutil.copyfileobj(f_in, f_out)

    # Delete original gzipped file
    logging.debug(f"Deleting {output_file}.gz")
    os.remove(f"{output_file}.gz")

    logging.info(f"\"{file_name}\" saved successfully!")


################
# Unity Assets #
################


def extract_object(resource_file, obj_type, obj_name=None, ext=None):

    if obj_name is not None:
        logging.info(f"Attempting to extract \"{obj_name}\" ({obj_type}) in {resource_file}")
    else:
        logging.info(f"Attempting to extract all {obj_type} objects in {resource_file}")

    relative_resource_file = str(relative_dir(f"temp/{resource_file}"))

    am = AssetsManager(relative_resource_file)
    for asset in am.assets.values():
        for obj in asset.objects.values():

            if str(obj.type) != str(obj_type):
                continue

            data = obj.read()

            # if we're looking for a specific object by name
            if obj_name is not None and data.name != obj_name:
                continue

            save_resource(data.name, obj.type, data, ext)
            # return


def save_resource(file_name, obj_type, data, ext):

    # Guess extension if it wasn't defined
    if ext is None:
        if obj_type == "TextAsset":
            first_line = data.text.partition("\n")[0]
            if "xml" in first_line:
                ext = "xml"
            elif "{" in first_line or "[" in first_line:
                ext = "json"
            else:
                ext = "txt"

    rel_path = relative_dir(f"output/current/{ext}")
    create_dir(rel_path)
    output_file = rel_path / f"{file_name}.{ext}"

    if obj_type == "TextAsset":
        text = data.text.replace("\n", "")
        save_text(rel_path, f"{file_name}.{ext}", text)
    elif obj_type == "Sprite":
        data.save(output_file)

    logging.info(f"Successfully exported {output_file}")


def do_manifest_stuff():
    logging.info(f"Merging xml files...")
    rel_dir = relative_dir("./output/current/json/manifest.json")
    manifest = read_json(rel_dir)

    # ignore this key as it has no paths
    del manifest["settings"]

    for merge_file in manifest:
        xml_files = []
        file_names = []
        for file in manifest[merge_file]:
            path = file["path"]
            if not path.endswith("xml"):
                continue

            file_name = ntpath.basename(path)
            rel_file = relative_dir(f"./output/current/xml/{file_name}")

            xml_files.append(rel_file)
            file_names.append(file_name)
            # data = read_file(rel_file)
            # xml = xmltodict.parse(data)

            # Get the first xml key
            # e.g. GroundTypes Objects EquipmentSets
            # key = next(iter(xml)) 

        if xml_files == []:
            continue

        logging.debug(f"Merging {file_names}")
        merged = merge_xml(xml_files)

        save_text(f"output/current/merged", f"{merge_file}.xml", merged)
        logging.info(f"Successfully merged {len(file_names)} xml files into {merge_file}.xml")


def main():

    temp_dir = relative_dir("temp")
    output_dir = relative_dir("output")

    create_dir(temp_dir)
    logging.basicConfig(
        filename=temp_dir / "log.txt",
        format="%(asctime)s %(message)s",
        datefmt="%m/%d/%Y %I:%M:%S %p",
        level="INFO"
        # level="DEBUG"
    )

    # Log to stout and log file
    logging.getLogger().addHandler(logging.StreamHandler(sys.stdout))

    build_hash = get_build_hash()
    build_url = f"https://rotmg-build.decagames.com/build-release/{build_hash}/rotmg-exalt-win-64"

    timestamp = str(math.floor(datetime.now().timestamp()))
    save_text(output_dir, "last_updated.txt", timestamp)

    # Build hash check
    current_build_hash = output_dir / "current/build_hash.txt"
    if os.path.isfile(current_build_hash):
        current_build_hash = read_file(current_build_hash)
    
        if build_hash == current_build_hash:
            logging.info("Build Hash is equal, aborting.")
            return

    save_text(output_dir / "current", "build_hash.txt", build_hash)

    download_asset(build_url, "/RotMG%20Exalt_Data/", "resources.assets")
    # download_asset(build_url, "/RotMG%20Exalt_Data/", "resources.assets.resS")
    # download_asset(build_url, "/RotMG%20Exalt_Data/", "resources.resource")
    # download_asset(build_url, "/RotMG%20Exalt_Data/il2cpp_data/Metadata/", "global-metadata.dat")

    # extract_object("resources.assets", "TextAsset", "objects", "xml")
    # extract_object("resources.assets", "TextAsset", "ground", "xml")

    extract_object("resources.assets", "TextAsset")

    do_manifest_stuff()

    logging.info("Done! Copying log and directory")
    shutil.copyfile(relative_dir("./temp/log.txt"), relative_dir("./output/current/log.txt"))
    shutil.copytree(relative_dir("./output/current"), relative_dir(f"./output/{build_hash}"))


if __name__ == "__main__":
    main()
