import pathlib

# URL Hosts
PROD_URL = "https://www.realmofthemadgod.com"
TESTING_URL = "https://test.realmofthemadgod.com"

# URL Paths
APP_INIT_PATH = f"/app/init?platform=standalonewindows64&key=9KnJFxtTvLu2frXv"

# File Paths (absolute)
SRC_DIR = pathlib.Path(__file__).parent.parent

OUTPUT_DIR = SRC_DIR / "output"     # Public files on the webserver

TEMP_DIR =  SRC_DIR / "temp"        # Temp directory (cleared)
FILES_DIR = TEMP_DIR / "files"      # Used to download files to
WORK_DIR =  TEMP_DIR / "current"    # Used to generate output files before copying to `OUTPUT_DIR`

LAUNCHER_UNPACKER_WINDOWS = SRC_DIR / "launcher_unpacker" / "unpacker_win.exe"
LAUNCHER_UNPACKER_LINUX = SRC_DIR / "launcher_unpacker" / "unpacker_linux"