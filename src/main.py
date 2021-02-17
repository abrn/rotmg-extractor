import os
import math
from datetime import datetime
from time import sleep

from classes import AppSettings
from classes import logger
from classes import Constants
from functions import *


def extract_build(prod_name, build_name, app_settings):
    """ Download and extract a production or testing, client or launcher
    
    Params
    prod_name       -- "Production" or "Testing"
    build_name      -- "Client" or "Launcher"
    app_settings    -- An object containing the build's AppSettings' (build id/hash/version/cdn)
    """

    work_dir: Path = Constants.WORK_DIR / prod_name.lower() / build_name.lower()
    files_dir: Path = Constants.FILES_DIR / prod_name.lower() / build_name.lower()
    repo_dir: Path = Constants.REPO_DIR / prod_name.lower() / build_name.lower()

    logger.log(logging.INFO, f"{prod_name} {build_name}")
    IndentFilter.level += 1
    
    if not app_settings["build_hash"]:
        logger.log(logging.WARNING, f"{prod_name} does not have a client build available.")
        IndentFilter.level -= 1
        return

    # Compare build hashes
    build_hash_file = repo_dir / "build_hash.txt"
    if build_hash_file.is_file():
        current_build_hash = read_file(build_hash_file)
        if current_build_hash == app_settings["build_hash"]:
            logger.log(logging.INFO, f"Current build hash is equal, aborting.")
            IndentFilter.level -= 1
            return
    
    build_url = app_settings["build_cdn"] + app_settings["build_hash"] + "/" + app_settings["build_id"]
    logger.log(logging.INFO, f"Build URL is {build_url}")

    # Game assets are easily downloaded using checksum.json, however the
    # launcher's assets must be unpacked. This leads to different directories
    # when we need to extract assets later
    build_assets_dir = files_dir

    # Download the build's (unity) files
    if build_name == "Client":
        download_client_assets(build_url, files_dir)
    else:
        launcher_downloaded = download_asset(build_url, "", ".exe", files_dir, gz=False)
        if not launcher_downloaded:
            logger.log(logging.ERROR, f"{prod_name} has no launcher build available or we failed to download it! Aborting.")
            IndentFilter.level -= 1
            return

        # The only file on the cdn is RotMG-Exalt-Installer.exe (build_id + .exe)
        launcher_file = app_settings["build_id"] + ".exe"
        unpack_launcher_assets(files_dir / launcher_file, files_dir)

        # these directories are outputted by launcher_unpacker.exe
        build_assets_dir = files_dir / "launcher" / "programfiles" 
        if not build_assets_dir.exists():
            logger.log(logging.ERROR, "Failed to unpack launcher assets, aborting!")
            IndentFilter.level -= 1
            return


    archive_build_files(build_assets_dir, work_dir)

    extracted_assets_dir = work_dir / "extracted_assets"
    extract_unity_assets(build_assets_dir, extracted_assets_dir)

    # Build specific things to do afterwards
    if build_name == "Client":

        # Extract exalt version (e.g. 1.3.2.1.0)
        metadata_file = files_dir / "RotMG Exalt_Data" / "il2cpp_data" / "Metadata" / "global-metadata.dat"
        extract_exalt_version(metadata_file, work_dir / "exalt_version.txt")

        # Merge useful xml files (objects.xml, groundtypes.xml)
        merge_xml_files(extracted_assets_dir / "TextAsset" / "manifest.json", extracted_assets_dir, work_dir)

    logger.log(logging.INFO, f"Done extracting {prod_name} {build_name}")
    IndentFilter.level -= 1

    timestamp = math.floor(datetime.now().timestamp())
    write_file(work_dir / "timestamp.txt", str(timestamp))

    # TODO:
    # del prev and  Copy to repo


def main():

    # Delete previous contents of ./temp/
    shutil.rmtree(Constants.TEMP_DIR, ignore_errors=True)
    sleep(5)  # Wait for filesystem to catch up / prevent bugs

    # Setup logger
    logger.setup()

    prod_app_settings = AppSettings(Constants.PROD_URL)
    extract_build("Production", "Client", prod_app_settings.client)
    extract_build("Production", "Launcher", prod_app_settings.launcher)

    test_app_settings = AppSettings(Constants.TESTING_URL)
    extract_build("Testing", "Client", test_app_settings.client)
    extract_build("Testing", "Launcher", test_app_settings.launcher)

    logger.log(logging.INFO, "Done!")

if __name__ == "__main__":
    main()
