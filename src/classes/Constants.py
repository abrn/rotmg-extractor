import pathlib

#############
# URL Hosts #
#############
ROTMG_URLS = {
    "Production": "https://www.realmofthemadgod.com",
    "Testing": "https://test.realmofthemadgod.com"
}

#############
# URL Paths #
#############
APP_INIT_PATH = f"/app/init?platform=standalonewindows64&key=9KnJFxtTvLu2frXv"


##############
# File Paths #
##############

# ./src
SRC_DIR = pathlib.Path(__file__).parent.parent

# ./ - repository root
ROOT_DIR = SRC_DIR.parent

# ./output
OUTPUT_DIR = ROOT_DIR / "output"

# ./output/web - public files on the webserver
WEB_DIR = OUTPUT_DIR / "web"

# ./output/repo - git repository to automatically commit build updates to, only contains current builds
REPO_DIR = OUTPUT_DIR / "repo"

# ./output/web - Public files on the webserver
OUTPUT_DIR = ROOT_DIR / "output"     

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

LAUNCHER_UNPACKER_WINDOWS = BINARIES_DIR / "launcher_unpacker" / "windows" / "unpacker_win.exe"
LAUNCHER_UNPACKER_LINUX = BINARIES_DIR / "launcher_unpacker" / "linux" / "unpacker_linux.exe"

IL2CPP_DUMPER_WINDOWS = BINARIES_DIR / "il2cppdumper" / "windows" / "Il2CppDumper.exe"
IL2CPP_DUMPER_LINUX = BINARIES_DIR / "il2cppdumper" / "linux" / "Il2CppDumper"