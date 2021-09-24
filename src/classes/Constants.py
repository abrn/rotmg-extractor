import pathlib
from os.path import join

##### APPENGINE ENDPOINTS #####

ROTMG_URLS = {
    "Production": "https://realmofthemadgod.appspot.com",
    "Testing":    "https://rotmgtesting.appspot.com",
    "Testing2":   "https://realmtesting2.appspot.com",
    "Testing3":   "https://rotmgtesting3.appspot.com",
    "Testing4":   "https://rotmgtesting4.appspot.com",
    "Testing5":   "https://rotmgtesting5.appspot.com"
}


##### OUTPUT SETTINGS #####
#TODO: move this to a config file
WEBSERVER_URL = "https://rotmg.dev/"
# Add a webhook URL + ping role ID to send a Discord message when a new build is available
DISCORD_WEBHOOK_URL = ""
DISCORD_WEBHOOK_ROLE = ""


##### URL SETTINGS #####
APP_INIT_PATH = "/app/init?platform=standalonewindows64&key=9KnJFxtTvLu2frXv"


##### PATH SETTINGS #####
# ./src
SRC_DIR = pathlib.Path(__file__).parent.parent
# ./ - repository root
ROOT_DIR = SRC_DIR.parent
# ./output - all files, including temp outputted by the program
OUTPUT_DIR = ROOT_DIR / "output"
# ./output/publish - published outputs visible on the web server
PUBLISH_DIR = OUTPUT_DIR / "publish"
# ./output/temp - temporary directory cleared everytime the program is run
TEMP_DIR = OUTPUT_DIR / "temp"
# ./output/temp/files - temporary file download location
FILES_DIR = TEMP_DIR / "files"
# ./output/temp/work - temporary location to generate output before being copied to web/repo
WORK_DIR = TEMP_DIR / "work"


##### TOOL BINARIES #####
BINARIES_DIR = SRC_DIR / "binaries"

LAUNCHER_UNPACKER_WIN = BINARIES_DIR / "launcher_unpacker" / "unpacker-win.exe"
LAUNCHER_UNPACKER_UNIX = BINARIES_DIR / "launcher_unpacker" / "unpacker-linux"
LAUNCHER_UNPACKER_MAC = BINARIES_DIR / "launcher_unpacker" / "unpacker-linux"

IL2CPP_INSPECTOR_WIN = join(BINARIES_DIR, 'il2cpp_inspector', 'Il2CppInspector-cli-win.exe')
IL2CPP_INSPECTOR_UNIX = join(BINARIES_DIR, 'il2cpp_inspector', 'Il2CppInspector-linux')
IL2CPP_INSPECTOR_MAC = join(BINARIES_DIR, 'il2cpp_inspector', 'Il2CppInspector')

IL2CPP_DUMPER_WIN = join(BINARIES_DIR, 'il2cpp_dumper', 'Il2CppDumper.exe')
IL2CPP_DUMPER_UNIX = join(BINARIES_DIR, 'il2cpp_dumper', 'Il2CppDumper')
IL2CPP_DUMPER_MAC = join(BINARIES_DIR, 'il2cpp_dumper', 'Il2CppDumper')

CPPTOIL_WIN = join(BINARIES_DIR, 'cpp2il', 'cpp2il.exe')
CPPTOIL_UNIX = join(BINARIES_DIR, 'cpp2il', 'cpp2il')
CPPTOIL_MAC = join(BINARIES_DIR, 'cpp2il', 'cpp2il')
