import os
import shutil

from pathlib import Path
from time import sleep

from classes import AppSettings
from classes import logger
from classes import Constants
from functions import *

# Move main code to a function
# So we can do:
# foo(Constants.PROD_URL, Constants.WORK_DIR / "production")
# foo(Constants.TESTING_URL, Constants.WORK_DIR / "testing")
#
# directories:
# ./temp/current/production/client
# ./temp/current/production/launcher
# ./temp/current/testing/client
# ./temp/current/testing/launcher

# maybe retain directory structure in checksum.json
# TODO: duplicate file check! - make a function in File.py for this
#

def extract(cdn_url, work_dir):
    # TODO: move main function code here, so we can do the same things for
    # production testing
    pass


def main():

    # Delete previous contents of ./temp/
    shutil.rmtree(Constants.TEMP_DIR, ignore_errors=True)

    # Wait for filesystem to catch up / prevent bugs
    sleep(5)

    # Ensure required paths exist
    Path(Constants.FILES_DIR).mkdir(parents=True, exist_ok=True)
    Path(Constants.WORK_DIR).mkdir(parents=True, exist_ok=True)
    Path(Constants.OUTPUT_DIR).mkdir(parents=True, exist_ok=True)

    logger.setup()

    # Get Build cdn/hash/version/id from app/init (xml AppSettings)
    appSettings = AppSettings(Constants.PROD_URL)

    if appSettings.build_hash:
        logger.log(logging.INFO, f"Build Hash is {appSettings.build_hash}")
        write_file(Constants.WORK_DIR / "build_hash.txt",
                   appSettings.build_hash)
    else:
        logger.log(logging.ERROR, "Couldn't get build hash, aborting!")
        return

    # Compare build hash
    build_hash_file = Constants.OUTPUT_DIR / "current" / "build_hash.txt"
    if os.path.isfile(build_hash_file):
        current_build_hash = read_file(build_hash_file)
        if current_build_hash == appSettings.build_hash:
            logger.log(logging.INFO, f"Current build hash is equal, aborting.")
            return

    build_url = appSettings.build_cdn + \
        appSettings.build_hash + "/" + appSettings.build_id
    logger.log(logging.INFO, f"Build URL is {build_url}")

    # Download all assets
    download_all_assets(build_url)

    # Archive assets
    archive_build_assets()

    # Extract Unity Assets

    # Attempt to get version string, scuffed method
    metadata_file = Constants.FILES_DIR / "global-metadata.dat"
    version_string = get_version_string(metadata_file)

    if version_string is None:
        logging.error("Could not extract version string!")
        write_file(Constants.WORK_DIR / "exalt_version.txt", "")
    else:
        logger.log(logging.INFO, f"Version string is {version_string}")
        write_file(Constants.WORK_DIR / "exalt_version.txt", version_string)

    # Regex patterns to match files to extract assets from
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

    extract_all_assets(file_patterns, ignored_exts)


if __name__ == "__main__":
    main()
