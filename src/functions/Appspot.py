import logging
import requests
import xml.etree.ElementTree as ET
from time import sleep
from classes import Constants
from classes import logger
from classes.CustomLogger import IndentFilter
from functions.File import write_file

def string_to_xml(string: str):
    element = ET.fromstring(string)
    tree = ET.ElementTree(element)
    return tree


def do_appspot_request(url: str, params=None):
    response = requests.get(url, params).text
    if response.startswith("<Error"):
        logger.log(logging.ERROR, f"Recieved error when requesting URL: {url}")
        IndentFilter.level += 1
        logger.log(logging.ERROR, response)
        IndentFilter.level -= 1
        return (response, False)

    return (response, True)


def archive_appspot(base_url: str, build_name: str):

    params = {
        "guid": Constants.APPSPOT_GUID,
        "password": Constants.APPSPOT_PASSWORD,
        "clientToken": "44131b17a8aea5da6e6f4c24be9dce459867c08d",
        "game_net": "Unity",
        "play_platform": "Unity",
        "game_net_user_id": "",
    }
    params.update(params)
    
    account_verify, success = do_appspot_request(base_url + Constants.APPSPOT_ACCOUNT_VERIFY, params)
    if not success: return

    xml = string_to_xml(account_verify).getroot()
    params.update({
        "accessToken": xml.find("AccessToken").text
    })

    # del params["guid"]
    # del params["password"]

    verify_accesstoken, success = do_appspot_request(base_url + Constants.APPSPOT_VERIFY_ACCESSTOKEN, params)
    if not success: return

    params.update(Constants.APPSPOT_PLATFORM_PARAMS)

    work_dir = Constants.WORK_DIR / "appspot" / build_name.lower()          # ./output/temp/work/appspot/production/
    publish_dir = Constants.PUBLISH_DIR / "appspot" / build_name.lower()    # ./output/publish/appspot/production/

    for appspot_path in Constants.APPSPOT_PATHS:
        path = appspot_path["path"]
        temp_params = params
        temp_params.update(appspot_path["params"])

        url = base_url + appspot_path["path"]
        file_path = work_dir / str(path[1:] + ".xml")
        
        response, success = do_appspot_request(url, params)
        write_file(file_path, response, rename_duplicate=False, overwrite=True)
        # sleep(0.1)

    # TODO: if there is a current, diff it with work_dir
    # if different, publish to {timestamp}
    # and update /current

