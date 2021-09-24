
import urllib
import shutil
import ntpath
import gzip
import zipfile
import logging
import os

from time import time
from urllib.error import HTTPError
from pathlib import Path
from classes import logger, IndentFilter
from functions.ExtractAssets import unpack_launcher_assets
from .File import read_json


def download_asset(
    build_url: str, 
    url_path: str, 
    file_name: str, 
    output_path, zip=True) -> bool:
    """ Download a build files with a given build URL and path 
    Args:
        build_url (str): the URL hosting the build file
        url_path (str): the path where the file is stored at the URL
        file_name (str): the file name to use when saving assets 
        output_path (str): [optional] supply a custom path to store the downloaded file
        zip (bool): [optional] set to True to utilize gzip compression after download
    Returns:
        bool: True if successful, False if download failed """
    extension = '.gz' if zip else extension = ''
    download_url = f'{build_url}{url_path}{file_name}{extension}'.replace('\n', '%20')

    # file doesn't have a name, only extension
    # e.g. the launcher's exe file is just {build_id}.exe
    if "." in file_name and Path(file_name).stem == file_name:
        file_name = Path(download_url).name

    # check if the file path exists, otherwise create it
    if not os.path.exists(output_path): 
        try:
            Path(output_path).mkdir(parents=True, exist_ok=True)
            output_file: Path = output_path / file_name
        except Exception as e:
            logger.log(f"Failed creating path '{output_path}', switching to current directory", logging.ERROR)
            # use the current directory + a timestamp for the build files e.g  'build-1632431840'
            output_path: Path = os.path.join(os.cwd(), f"build-{int(time.time())}")
            output_file: Path = os.path.join(output_path, file_name)

    # check if the file already exists
    if os.path.exists(os.path.join(output_path, file_name)):
        logger.log(f"File '{file_name}' already exists in build files - ignoring", logging.WARN)
        return
    
    try:
        # attempt to download the file
        logger.log(f"Downloading {download_url}", logging.DEBUG)
        urllib.request.urlretrieve(download_url, f"{output_file}{extension}")
    except HTTPError as e:
        # TODO: implement some sort of retry mechanism if the download fails
        logger.log(f"Error downloading '{download_url}': {e.code} {e.msg}", logging.ERROR)
        return False
    logger.log(f"Downloaded '{file_name}'")
    
    if zip:
        logger.log(f"Extracting gzip file '{file_name}{extension}'", logging.DEBUG)
        try:
            with gzip.open(f"{output_file}{extension}", 'rb') as zip_file:
                with open(output_file, 'wb') as out_file:
                    # copy the file to the output directory
                    shutil.copyfileobj(zip_file, out_file)
                    shutil.remove(f"{output_file}{extension}")
        except ValueError as e:
            #TODO: implement retry mechanism for downloading zip file
            logger.log(f"Failed unzipping launcher file: zip file is corrupted". logging.ERROR)
            return False
        except NotImplementedError as e:
            logger.log(f"Failed unzipping launcher file: zip file is encrypted or unsupported". logging.ERROR)
            return False

    #TODO: compare the filesize/checksum for the file to reduce errors while downloading
    return output_file.exists()


def download_client_assets(build_url, output_path) -> str:
    """ Download a client build from a given URL to an output folder
    Args:
        build_url (str): the url of the client build
        output_path (str): the path to download the client build files to
    Returns:
        str: returns the output path of the downloaded files
    """
    logger.log("Downloading client build assets...")
    IndentFilter.level += 1

    # download the checksum.json file and parse the json data
    checksum_file = os.path.join(output_path, 'checksum.json')
    download_asset(build_url, '/', 'checksum.json', output_path, gzip=False)
    checksum_data = read_json(checksum_file)

    # loop through each file in the build checksum.data and download it
    for file in checksum_data['files']:
        file_name = ntpath.basename(file['file'])
        file_dir = ntpath.dirname(file['file'])

        # get the downloaded file name relative to the output path
        output_file_dir = os.path.join(output_path, file_dir)
        file_dir = '/' if file_dir == '' else f'/{file_dir}/'

        # start the download of the file
        download_asset(build_url, file_dir, file_name, output_file_dir, gzip=True)
    
    IndentFilter.level -= 1
    return output_path


def download_launcher_assets(build_url: str, build_id: str, output_path) -> Path or None:
    """ Attemps to download and extract the launcher build files and assets
    Args:
        build_url (str): the URL of the launcher build
        build_id (str): the launcher build ID
        output_path ([type]): the path to download the build files to
    Returns:
        Path or None: the output path if successful or None on failure
    """
    logger.log("Attempting to download the launcher installer exe")
    IndentFilter.level += 1

    # attempt to download the launcher installer
    if download_asset(build_url, '', '.exe', output_path, gzip=False):
        IndentFilter.level -= 1
        # unpack the launcher assets
        launcher_file = build_id + '.exe'
        unpack_launcher_assets(os.path.join(output_path, launcher_file), output_path)
        return os.path.join(output_path, 'launcher', 'programfiles')
    IndentFilter.level -= 1

    # attempt to download the launcher's assets (.zip) - used for testing launcher
    logger.log("Attempting to download launcher .zip file")
    IndentFilter.level += 1

    if download_asset(build_url, '', '.zip', output_path, gzip=False):
        files_dir = os.path.join(output_path, 'files_dir')
        # extract the launcher zip file
        with zipfile.ZipFile(os.path.join(output_path, build_id, '.zip'), 'r') as zip_ref:
            zip_ref.extractall(files_dir)        
        logger.log("Extracted launcher .zip file")
        IndentFilter.level -= 1
        return os.path.join(files_dir)

    #TODO: retry here using @requests instead of failing once?
    IndentFilter.level -= 1
    logger.log(f"Failed to download a launcher build...", logging.ERROR)
    logger.log(f"Build URL: {build_url}\nBuild ID: {build_id}", logging.ERROR)
    return None
