import os
import math
import shutil
import gzip
import urllib.request
import xmltodict
from pathlib import Path
import logging
from UnityPy import AssetsManager
from datetime import datetime

###################
# File Operations #
###################


def create_dir(path):
    Path(path).mkdir(parents=True, exist_ok=True)


def write_file(dir_path, file_name, text):
    create_dir(dir_path)
    output_file = f"{dir_path}/{file_name}"
    with open(output_file, "w") as file:
        file.write(text)


def read_file(file_path):
    with open(file_path) as file:
        return file.read()


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


def download_build_asset(build_url, path, file_name):
    url = build_url + path + file_name + ".gz"

    # Create directory if not exist
    create_dir("./temp")
    output_file = f"./temp/{file_name}"

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


def extract_resources(build_hash, resource_file, obj_type, data_name, ext):
    output_dir = f"./output/{build_hash}/"
    logging.debug(f"Extracting {data_name} ({obj_type}) from {resource_file}")
    am = AssetsManager(f"./temp/{resource_file}")
    for asset in am.assets.values():
        for obj in asset.objects.values():
            if obj.type == obj_type:
                data = obj.read()
                if data.name == data_name:
                    # Create directory if not exist
                    Path(output_dir).mkdir(parents=True, exist_ok=True)

                    save_resource(f"./output/{build_hash}", data.name, ext, obj.type, data)
                    save_resource(f"./output/current", data.name, ext, obj.type, data)
                    logging.info(f"Successfully exported {data_name}.{ext}")
                    return


def save_resource(dir_path, file_name, ext, obj_type, data):
    # Create dir if not exist
    create_dir(dir_path)
    output_file = f"{dir_path}/{file_name}.{ext}"

    if obj_type == "TextAsset":
        text = data.text.replace("\n", "")
        write_file(dir_path, f"{file_name}.{ext}", text)
    elif obj_type == "Sprite":
        data.save(output_file)


def main():
    create_dir("./temp/")
    logging.basicConfig(filename="./temp/log.txt",
                        format="%(asctime)s %(message)s",
                        datefmt="%m/%d/%Y %I:%M:%S %p",
                        level="INFO")

    build_hash = get_build_hash()
    build_url = f"https://rotmg-build.decagames.com/build-release/{build_hash}/rotmg-exalt-win-64"

    timestamp = str(math.floor(datetime.now().timestamp()))
    write_file("./output", "last_updated.txt", timestamp)

    current_build_hash = "./output/current/build_hash.txt"
    if os.path.isfile(current_build_hash):
        current_build_hash = read_file(current_build_hash)

        if build_hash == current_build_hash:
            logging.info("Build Hash is equal, aborting.")
            return

    write_file("./output/current", "build_hash.txt", build_hash)

    download_build_asset(build_url, "/RotMG%20Exalt_Data/", "resources.assets")
    # download_build_asset(build_url, "/RotMG%20Exalt_Data/", "resources.assets.resS")
    # download_build_asset(build_url, "/RotMG%20Exalt_Data/", "resources.resource")

    extract_resources(build_hash, "resources.assets", "TextAsset", "objects", "xml")
    extract_resources(build_hash, "resources.assets", "TextAsset", "ground", "xml")

    # Copy log.txt once finished
    shutil.copyfile("./temp/log.txt", "./output/current/log.txt")
    shutil.copyfile("./temp/log.txt", f"./output/{build_hash}/log.txt")


if __name__ == "__main__":
    main()
