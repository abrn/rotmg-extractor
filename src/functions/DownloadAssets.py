
import os
import urllib
from urllib.error import HTTPError
import shutil
import ntpath
import gzip
import logging
from pathlib import Path

from classes import Constants
from classes import logger, IndentFilter
from .File import read_json


def download_asset(build_url, url_path, file_name, output_path, gz=True):
    """
    Downloads a build asset, automatically extracting the file if it was gzipped.

    Paramaters
    build_url   -- The url of the CDN to use (example `AppSettings.BuildCDN`)
    url_path    -- The url path to the asset, excluding the filename
    file_name   -- The file name
    output_path  -- The output directory of the file. Default is "./temp"
    gz          -- If the file is stored on the CDN as a gzipped archive, and should be extracted. Default is True
    """

    # TODO: Retain filepaths

    ext = ""
    if gz:
        ext = ".gz"

    download_url = build_url + url_path + file_name + ext
    download_url = download_url.replace(" ", "%20")

    # file doesn't have a name, only extension
    if "." in file_name and Path(file_name).stem == file_name:
        file_name = Path(download_url).name

    Path(output_path).mkdir(parents=True, exist_ok=True)
    output_file = output_path / file_name

    logger.log(logging.DEBUG, f"Downloading {download_url}")

    try:
        urllib.request.urlretrieve(download_url, f"{output_file}{ext}")
    except HTTPError as e:
        logger.log(logging.ERROR, f"Error downloading \"{download_url}\". Error: {e.code} {e.msg}")
        return False

    if gz:
        logger.log(logging.DEBUG, f"Extracting {file_name}{ext}")
        with gzip.open(f"{output_file}{ext}", "rb") as f_in:
            with open(output_file, "wb") as f_out:
                shutil.copyfileobj(f_in, f_out)

        os.remove(f"{output_file}{ext}")

    logger.log(logging.INFO, f"Downloaded {file_name}")
    return True

def download_client_assets(build_url, output_path):
    """ Downloads and extracts all the client assets """

    logger.log(logging.INFO, "Downloading all client build assets...")

    checksum_file = output_path / "checksum.json"
    download_asset(build_url, "/", "checksum.json", output_path, gz=False)
    checksum_data = read_json(checksum_file)

    IndentFilter.level += 1
    for file in checksum_data["files"]:
        file_name = ntpath.basename(file["file"])
        file_dir = ntpath.dirname(file["file"])

        if file_dir == "":
            file_dir = "/"
        else:
            file_dir = "/" + file_dir + "/"

        download_asset(build_url, file_dir, file_name, output_path, gz=True)

    IndentFilter.level -= 1