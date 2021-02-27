import shutil
import math
from datetime import datetime
from time import sleep

from classes import AppSettings
from classes import logger
from classes import Constants
from functions import *


def full_build_extract(prod_name, build_name, app_settings):
    work_dir: Path = Constants.WORK_DIR / prod_name.lower() / build_name.lower()
    files_dir: Path = Constants.FILES_DIR / prod_name.lower() / build_name.lower()
    repo_dir: Path = Constants.REPO_DIR / build_name.lower()

    logger.log(logging.INFO, f"Starting {prod_name} {build_name}")
    IndentFilter.level += 1

    pre_setup = pre_build_setup(prod_name, build_name, app_settings, work_dir, repo_dir)
    if not pre_setup:
        return False

    build_files_dir = download_archive_build(prod_name, build_name, app_settings, files_dir, work_dir)
    if not build_files_dir:
        return False
    
    extracted = extract_build(build_name, build_files_dir, work_dir)
    if not extracted:
        return False

    output_build(prod_name, build_name, app_settings, work_dir, repo_dir, extracted[0])

    logger.log(logging.INFO, f"Done {prod_name} {build_name}")
    IndentFilter.level -= 1


def pre_build_setup(prod_name, build_name, app_settings, work_dir, repo_dir):

    if not app_settings["build_hash"]:
        logger.log(logging.WARNING, f"{prod_name} does not have a {build_name} build available, aborting.")
        IndentFilter.level -= 1
        return False

    if prod_name.lower() in repo.remotes.origin.refs:
        # branch exists, switch to it
        logger.log(logging.INFO, f"Switching to branch \"{prod_name.lower()}\"")
        repo.git.checkout(prod_name.lower())
    else:
        # branch doesn't exist, create an orphan branch (and switch to it)
        logger.log(logging.INFO, f"Created branch \"{prod_name.lower()}\"")
        repo.git.checkout("--orphan", prod_name.lower())
        delete_dir_contents(Constants.REPO_DIR)

    # Compare build hashes
    build_hash_file = repo_dir / "build_hash.txt"
    if build_hash_file.is_file():
        current_build_hash = read_file(build_hash_file)
        if current_build_hash == app_settings["build_hash"]:
            logger.log(logging.INFO, f"Current build hash is equal, aborting.")
            IndentFilter.level -= 1
            return False

    write_file(work_dir / "build_hash.txt", app_settings["build_hash"], overwrite=True)
    return True


def download_archive_build(prod_name, build_name, app_settings, files_dir, work_dir, download=True, archive=True):

    build_url = app_settings["build_cdn"] + app_settings["build_hash"] + "/" + app_settings["build_id"]
    logger.log(logging.INFO, f"Build URL is {build_url}")

    # Download build files, output directory can change depending 
    # if it's the client vs how the launcher exe is unpacked 
    build_files_dir = None

    if download:
        if build_name == "Client":
            build_files_dir = download_client_assets(build_url, files_dir / "files_dir")
        elif build_name == "Launcher":
            build_files_dir = download_launcher_assets(build_url, app_settings["build_id"], files_dir)

    if build_files_dir is None:
        logger.log(logging.ERROR, f"Failed to download/extract {prod_name} {build_name} assets! Aborting")
        return False
    
    if archive:
        archive_build_files(build_files_dir, work_dir)

    return build_files_dir


def extract_build(build_name, build_files_dir, work_dir):

    extracted_assets_dir = work_dir / "extracted_assets"
    extract_unity_assets(build_files_dir, extracted_assets_dir)

    exalt_version = ""
    if build_name == "Client":
        # Extract exalt version (e.g. 1.3.2.1.0)
        metadata_file = build_files_dir / "RotMG Exalt_Data" / "il2cpp_data" / "Metadata" / "global-metadata.dat"
        exalt_version = extract_exalt_version(metadata_file, work_dir / "exalt_version.txt")

        merge_xml_files(extracted_assets_dir / "TextAsset" / "manifest.json", extracted_assets_dir, work_dir)

    # Dump GameAssembly using Il2CppDumper
    data_dir = find_path(build_files_dir, "*_Data")
    metadata = data_dir / "il2cpp_data" / "Metadata" / "global-metadata.dat"
    gameassembly = build_files_dir / "GameAssembly.dll"
    dump_output = work_dir / "il2cpp_dump"
    dump_il2cpp(gameassembly, metadata, dump_output)
    
    return (exalt_version,)


def output_build(prod_name, build_name, app_settings, work_dir, repo_dir, exalt_version=""):

    logger.log(logging.INFO, "Outputting build...")
    IndentFilter.level += 1

    timestamp = math.floor(datetime.now().timestamp())
    write_file(work_dir / "timestamp.txt", str(timestamp))

    # Move files to repo
    logger.log(logging.INFO, f"Deleting {repo_dir}")
    shutil.rmtree(repo_dir, ignore_errors=True)
    sleep(2)

    logger.log(logging.INFO, f"Copying work_dir to repo_dir")
    shutil.copytree(work_dir, repo_dir)
    sleep(2)

    if build_name == "Client":
        commit_new_build(prod_name, build_name, app_settings, exalt_version)
    elif build_name == "Launcher":
        commit_new_build(prod_name, build_name, app_settings)

    IndentFilter.level -= 1
    return True


def main():

    # Delete previous contents of ./temp/
    shutil.rmtree(Constants.TEMP_DIR, ignore_errors=True)
    sleep(5) # Wait for filesystem to catch up / prevent bugs

    # Setup logger
    logger.setup()

    # build_commits = get_build_commits()
    # append_commits(build_commits, "Extracted MonoScript")

    prod_names = ["Production", "Testing"]
    for prod_name in prod_names:
        app_settings = AppSettings(Constants.ROTMG_URLS["Production"])
        full_build_extract(prod_name, "Client", app_settings.client)
        full_build_extract(prod_name, "Launcher", app_settings.launcher)

    logger.log(logging.INFO, "Done!")

    # TODO: loop main here


if __name__ == "__main__":
    main()