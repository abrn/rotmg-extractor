import os
from time import sleep

from classes import AppSettings
from classes import logger
from classes import Constants
from functions import *

def extract(prod_name, app_settings: AppSettings):

    # Production -> production
    output_dir = prod_name.strip().lower()
    work_dir = Constants.WORK_DIR / output_dir
    files_dir = Constants.FILES_DIR / output_dir

    # Save app_settings to ./current/prod/app_settings.xml
    write_file(
        file_path=work_dir / "app_settings.xml",
        data=app_settings.xml,
        mode="wb",
        overwrite=True
    )

    # Extract client, launcher
    extract_client(prod_name, app_settings.client, files_dir / "client", work_dir / "client")
    extract_launcher(prod_name, app_settings.launcher, files_dir / "launcher", work_dir / "launcher")
    
    # TODO:
    # git commit?
    # generate diff (text and html)


def extract_client(prod_name, app_settings, files_dir, work_dir):
    """ download and extract the latest production/testing client build """
    
    if not app_settings["build_hash"]:
        logger.log(logging.WARNING, f"{prod_name} Client does not have a build available.")
        return

    logger.log(logging.INFO, f"Extracting {prod_name} Client")
    IndentFilter.level += 1

    logger.log(logging.INFO, f"Build Hash is {app_settings['build_hash']}")
    write_file(work_dir / "build_hash.txt", app_settings["build_hash"], overwrite=True)

    # Compare build hash
    build_hash_file = Constants.OUTPUT_DIR / "current" / prod_name / "client" / "build_hash.txt"
    if os.path.isfile(build_hash_file):
        current_build_hash = read_file(build_hash_file)
        if current_build_hash == app_settings["build_hash"]:
            logger.log(logging.INFO, f"Current build hash is equal, aborting.")
            IndentFilter.level -= 1
            return

    build_url = app_settings["build_cdn"] + app_settings["build_hash"] + "/" + app_settings["build_id"]
    logger.log(logging.INFO, f"Build URL is {build_url}")

    # Download all build assets
    download_asset(build_url, "/", "checksum.json", files_dir, gz=False)
    download_client_assets(build_url, files_dir)

    archive_build_assets(files_dir, work_dir)

    version_string = get_version_string(files_dir / "global-metadata.dat")
    if version_string is None:
        logging.error("Could not extract version string!")
        write_file(work_dir / "exalt_version.txt", "")
    else:
        logger.log(logging.INFO, f"Version string is {version_string}")
        write_file(work_dir / "exalt_version.txt", version_string)

    extract_all_assets(files_dir, work_dir / "unity_assets")

    manifest_file = work_dir / "unity_assets" / "TextAsset" / "manifest.json"
    if not os.path.isfile(manifest_file):
        logger.log(logging.ERROR, f"{prod_name} Client has no manifest.json!")
    else:
        merge_xml_files(manifest_file, work_dir / "unity_assets", work_dir)

    IndentFilter.level -= 1

    pass

def extract_launcher(prod_name, app_settings, files_dir, work_dir):
    # https://rotmg-build.decagames.com/launcher-release/d554e291899750f9d36c750798e85646/RotMG-Exalt-Installer.exe
    
    if not app_settings["build_hash"]:
        logger.log(logging.WARNING, f"{prod_name} Launcher does not have a build available.")
        return

    logger.log(logging.INFO, f"Extracting {prod_name} Launcher")
    IndentFilter.level += 1
    # TODO
    IndentFilter.level -= 1


def main():

    # Delete previous contents of ./temp/
    shutil.rmtree(Constants.TEMP_DIR, ignore_errors=True)
    sleep(5) # Wait for filesystem to catch up / prevent bugs

    # Setup logger
    logger.setup()

    prod_app_settings = AppSettings(Constants.PROD_URL)
    extract("Production", prod_app_settings)

    test_app_settings = AppSettings(Constants.TESTING_URL)
    extract("Testing", test_app_settings)


if __name__ == "__main__":
    main()
