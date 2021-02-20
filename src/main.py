import shutil
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
    repo_dir: Path = Constants.REPO_DIR / build_name.lower()

    logger.log(logging.INFO, f"{prod_name} {build_name}")
    IndentFilter.level += 1
    
    if prod_name.lower() in repo.references:
        # branch exists, switch to it
        logger.log(logging.INFO, f"Switching to branch \"{prod_name.lower()}\"")
        repo.git.checkout(prod_name.lower())
    else:
        # branch doesn't exist, create an orphan branch (and switch to it)
        logger.log(logging.INFO, f"Created branch \"{prod_name.lower()}\"")
        repo.git.checkout("--orphan", prod_name.lower())
        delete_dir_contents(Constants.REPO_DIR)

    if not app_settings["build_hash"]:
        logger.log(logging.WARNING, f"{prod_name} does not have a {build_name} build available.")
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
    
    write_file(work_dir / "build_hash.txt", app_settings["build_hash"])

    build_url = app_settings["build_cdn"] + app_settings["build_hash"] + "/" + app_settings["build_id"]
    logger.log(logging.INFO, f"Build URL is {build_url}")

    # Game assets are easily downloaded using checksum.json, however the
    # launcher's assets may need to be unpacked. This leads to different directories
    # when we need to extract assets later
    build_files_dir = None

    # Download the build's files
    if build_name == "Client":
        build_files_dir = download_client_assets(build_url, files_dir / "files_dir")
    else:
        build_files_dir = download_launcher_assets(build_url, app_settings["build_id"], files_dir)

    if build_files_dir is None:
        logger.log(logging.ERROR, f"Failed to download/extract {prod_name} {build_name} assets! Aborting")
        return

    archive_build_files(build_files_dir, work_dir)

    extracted_assets_dir = work_dir / "extracted_assets"
    extract_unity_assets(build_files_dir, extracted_assets_dir)

    exalt_version = ""

    # Build specific things to do afterwards
    if build_name == "Client":

        # Extract exalt version (e.g. 1.3.2.1.0)
        metadata_file = build_files_dir / "RotMG Exalt_Data" / "il2cpp_data" / "Metadata" / "global-metadata.dat"
        exalt_version = extract_exalt_version(metadata_file, work_dir / "exalt_version.txt")

        # Merge useful xml files (objects.xml, groundtypes.xml)
        merge_xml_files(extracted_assets_dir / "TextAsset" / "manifest.json", extracted_assets_dir, work_dir)

    timestamp = math.floor(datetime.now().timestamp())
    write_file(work_dir / "timestamp.txt", str(timestamp))

    # Commit the changes to the repo
    shutil.rmtree(repo_dir, ignore_errors=True)
    sleep(2)
    shutil.copytree(work_dir, repo_dir)
    sleep(2)

    if build_name == "Client":
        commit_new_build(prod_name, build_name, app_settings, exalt_version)
    else:
        commit_new_build(prod_name, build_name, app_settings)

    logger.log(logging.INFO, f"Done {prod_name} {build_name}")
    IndentFilter.level -= 1


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
