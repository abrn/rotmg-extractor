
import os
import urllib
import shutil
import ntpath
import gzip
import zipfile
import logging
from urllib.error import HTTPError
from pathlib import Path

from classes import Constants
from classes import logger, IndentFilter
from functions.ExtractAssets import unpack_launcher_assets
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

    ext = ""
    if gz:
        ext = ".gz"

    download_url = build_url + url_path + file_name + ext
    download_url = download_url.replace(" ", "%20")

    # file doesn't have a name, only extension
    # e.g. the launcher's exe file is just {build_id}.exe
    if "." in file_name and Path(file_name).stem == file_name:
        file_name = Path(download_url).name

    Path(output_path).mkdir(parents=True, exist_ok=True)
    output_file: Path = output_path / file_name

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
    return output_file.exists()


def download_client_assets(build_url, output_path):
    """ Downloads all the client assets, automatically extracting gzipped files """

    logger.log(logging.INFO, "Downloading client build assets...")
    IndentFilter.level += 1

    checksum_file = output_path / "checksum.json"
    download_asset(build_url, "/", "checksum.json", output_path, gz=False)
    checksum_data = read_json(checksum_file)
    
    for file in checksum_data["files"]:
        file_name = ntpath.basename(file["file"])
        file_dir = ntpath.dirname(file["file"])

        # Retain directory structure
        output_file_dir = output_path / file_dir

        if file_dir == "":
            file_dir = "/"
        else:
            file_dir = "/" + file_dir + "/"

        download_asset(build_url, file_dir, file_name, output_file_dir, gz=True)

    IndentFilter.level -= 1
    return output_path


def download_launcher_assets(build_url, build_id, output_path):
    """ Attemps to download and extract the launcher's installer or build assets """

    # Attempt to download the launcher's installer (.exe)
    logger.log(logging.INFO, "Attempting to download launcher exe (installer)")
    IndentFilter.level += 1

    installer_downloaded = download_asset(build_url, "", ".exe", output_path, gz=False)
    if installer_downloaded:
        launcher_file = build_id + ".exe"
        unpack_launcher_assets(output_path / launcher_file, output_path)

        # outputted directories by the unpacker
        IndentFilter.level -= 1
        return output_path / "launcher" / "programfiles" 
    
    IndentFilter.level -= 1

    # Attempt to download the launcher's assets (.zip)
    logger.log(logging.INFO, "Attempting to download launcher zip")
    IndentFilter.level += 1

    assets_downloaded = download_asset(build_url, "", ".zip", output_path, gz=False)
    if assets_downloaded:
        assets_file = build_id + ".zip"
        # extract zip
        with zipfile.ZipFile(output_path / assets_file, "r") as zip_ref:
            zip_ref.extractall(output_path / "files_dir")
        
        IndentFilter.level -= 1
        return output_path / "files_dir"

    IndentFilter.level -= 1
    return None