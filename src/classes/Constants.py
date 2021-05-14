import pathlib

#############
# URL Hosts #
#############
ROTMG_URLS = {
    "Production": "https://www.realmofthemadgod.com",
    "Testing": "https://test.realmofthemadgod.com",
    "Testing2": "https://test2.realmofthemadgod.com",
    "Testing3": "https://test3.realmofthemadgod.com",
    "Testing4": "https://test4.realmofthemadgod.com"
    # "Testing5": "https://test5.realmofthemadgod.com"
}

#############
# URL Paths #
#############
APP_INIT_PATH = "/app/init?platform=standalonewindows64&key=9KnJFxtTvLu2frXv"


##############
# File Paths #
##############

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

############
# Binaries #
############

BINARIES_DIR = SRC_DIR / "binaries"

LAUNCHER_UNPACKER_WINDOWS = BINARIES_DIR / "launcher_unpacker" / "unpacker-win.exe"
LAUNCHER_UNPACKER_LINUX = BINARIES_DIR / "launcher_unpacker" / "unpacker-linux"

IL2CPP_DUMPER_WINDOWS = BINARIES_DIR / "Il2CppInspector" / "Il2CppInspector-cli-win.exe"
IL2CPP_DUMPER_LINUX = BINARIES_DIR / "Il2CppInspector" / "Il2CppInspector-linux"