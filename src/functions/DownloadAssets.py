
import os
import urllib
import shutil
import ntpath
import gzip
import logging

from classes import Constants
from classes import logger, IndentFilter
from .File import read_json


def download_asset(build_cdn, url_path, file_name, output_dir=Constants.FILES_DIR, gz=True):
    """
    Downloads a build asset, automatically extracting the file if it was gzipped.

    Paramaters
    build_cdn   -- The url of the CDN to use (example `AppSettings.BuildCDN`)
    url_path    -- The url path to the asset, excluding the filename
    file_name   -- The file name
    output_dir  -- The output directory of the file. Default is "./temp"
    gz          -- If the file is stored on the CDN as a gzipped archive, and should be extracted. Default is True
    """

    ext = ""
    if (gz):
        ext = ".gz"

    download_url = build_cdn + url_path + file_name + ext
    download_url = download_url.replace(" ", "%20")

    output_file = output_dir / file_name

    logger.log(logging.DEBUG, f"Downloading {download_url}")
    IndentFilter.level += 1

    urllib.request.urlretrieve(download_url, f"{output_file}{ext}")

    if gz:
        logger.log(logging.DEBUG, f"Extracting {file_name}{ext}")
        with gzip.open(f"{output_file}{ext}", "rb") as f_in:
            with open(output_file, "wb") as f_out:
                shutil.copyfileobj(f_in, f_out)

        os.remove(f"{output_file}{ext}")

    logger.log(logging.INFO, f"Downloaded {file_name}")
    IndentFilter.level -= 1


def download_all_assets(build_url, download_checksum=True):
    """ Downloads all the files present in the `checksum.json` file """

    logger.log(logging.INFO, "Downloading all build assets...")

    checksum_file = Constants.FILES_DIR / "checksum.json"

    if download_checksum:
        download_asset(build_url, "/", "checksum.json", gz=False)

    checksum_data = read_json(checksum_file)

    for file in checksum_data["files"]:
        file_name = ntpath.basename(file["file"])
        file_dir = ntpath.dirname(file["file"])

        if file_dir == "":
            file_dir = "/"
        else:
            file_dir = "/" + file_dir + "/"

        download_asset(build_url, file_dir, file_name, gz=True)

    logger.log(logging.INFO, "All assets downloaded!")
