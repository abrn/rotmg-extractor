import os
from time import sleep

from classes import AppSettings
from classes import logger
from classes import Constants
from functions import *
from functions.UnpackLauncher import unpack_launcher_assets


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

    # Extract client
    extract_client(
        prod_name,
        app_settings.client,
        files_dir / "client", work_dir / "client"
    )

    # Extract launcher
    extract_launcher(
        prod_name,
        app_settings.launcher,
        files_dir / "launcher", work_dir / "launcher"
    )

    # TODO:
    # git commit?
    # generate diff (text and html)


def extract_client(prod_name, app_settings, files_dir, work_dir):
    """ download and extract the latest production/testing client build """

    logger.log(logging.INFO, f"Extracting {prod_name} Client")
    IndentFilter.level += 1

    if not app_settings["build_hash"]:
        logger.log(logging.WARNING, f"{prod_name} does not have a client build available.")
        IndentFilter.level -= 1
        return

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


def extract_launcher(prod_name, app_settings, files_dir, work_dir):
    # https://rotmg-build.decagames.com/launcher-release/d554e291899750f9d36c750798e85646/RotMG-Exalt-Installer.exe

    if not app_settings["build_hash"]:
        logger.log(logging.WARNING, f"{prod_name} does not have a launcher build available.")
        return

    logger.log(logging.INFO, f"Extracting {prod_name} Launcher")
    IndentFilter.level += 1

    logger.log(logging.INFO, f"Build Hash is {app_settings['build_hash']}")
    write_file(work_dir / "build_hash.txt", app_settings["build_hash"], overwrite=True)

    # Compare build hash
    build_hash_file = Constants.OUTPUT_DIR / "current" / prod_name / "launcher" / "build_hash.txt"
    if os.path.isfile(build_hash_file):
        current_build_hash = read_file(build_hash_file)
        if current_build_hash == app_settings["build_hash"]:
            logger.log(logging.INFO, f"Current build hash is equal, aborting.")
            IndentFilter.level -= 1

    # Not really a "build url" to use, the only file on the launcher cdn is {build_id}.exe
    # E.g. https://rotmg-build.decagames.com/launcher-release/d554e291899750f9d36c750798e85646/RotMG-Exalt-Installer.exe
    build_url = app_settings["build_cdn"] + app_settings["build_hash"] + "/" + app_settings["build_id"]
    launcher_name = app_settings["build_id"] + ".exe"

    # Download RotMG-Exalt-Installer.exe
    launcher_downloaded = download_asset(build_url, "", ".exe", files_dir, gz=False)
    if not launcher_downloaded:
        logger.log(logging.ERROR, f"{prod_name} has no launcher build available or we failed to download it! Aborting.")
        return

    # Extract files from launcher There is no checksum.json to download all
    # build files from - so instead we must extract the files from .exe.
    unpack_launcher_assets(files_dir / launcher_name, files_dir.parent)

    archive_build_assets(files_dir, work_dir)

    extract_all_assets(files_dir / "programfiles" / "RotMG Exalt Launcher_Data", work_dir / "unity_assets")

    IndentFilter.level -= 1


def main():

    # Delete previous contents of ./temp/
    shutil.rmtree(Constants.TEMP_DIR, ignore_errors=True)
    sleep(5)  # Wait for filesystem to catch up / prevent bugs

    # Setup logger
    logger.setup()

    prod_app_settings = AppSettings(Constants.PROD_URL)
    extract("Production", prod_app_settings)

    test_app_settings = AppSettings(Constants.TESTING_URL)
    extract("Testing", test_app_settings)


if __name__ == "__main__":
    main()
