import pathlib

#############
# URL Hosts #
#############
APPSPOT_URLS = {
    "Production": "https://realmofthemadgod.appspot.com",
    "Testing":    "https://rotmgtesting.appspot.com",
    "Testing2":   "https://realmtesting2.appspot.com",
    "Testing3":   "https://rotmgtesting3.appspot.com",
    "Testing4":   "https://rotmgtesting4.appspot.com",
    "Testing5":   "https://rotmgtesting5.appspot.com"
}

APPSPOT_GUID = "qreuoteybpja@gmail.com"
APPSPOT_PASSWORD = "8fuZAnnHTLbj"
APPSPOT_PLATFORM_PARAMS = {"platform": "standalonewindows64", "key": "9KnJFxtTvLu2frXv"}

# Paths that are actually useful, need to use them in code
APPSPOT_APP_INIT = "/app/init"
APPSPOT_ACCOUNT_VERIFY = "/account/verify"
APPSPOT_VERIFY_ACCESSTOKEN = "/account/verifyAccessTokenClient"

APPSPOT_PATHS = [

    # /account/ #
    {
        "path": "/account/extendAccessToken",
        "params": {}
    },
    {
        "path": "/account/getOwnedPetSkins",
        "params": {}
    },
    {
        "path": "/account/getCredits",
        "params": {}
    },
    {
        "path": "/account/list",
        "params": {
            "type": 0
        }
    },
    {
        "path": "/account/listPowerUpStats",
        "params": {}
    },
    {
        "path": "/account/servers",
        "params": {}
    },
    {
        "path": APPSPOT_ACCOUNT_VERIFY,
        "params": {}
    },

    # /app/ #
    {
        "path": "/app/publicStaticData",
        "params": {
            "dataType": "powerUpSettings"
        }
    },
    {
        "path": "/app/globalNews",
        "params": {}
    },
    {
        "path": "/app/getLanguageStrings",
        "params": {
            "languageType": "en"
        }
    },
    {
        "path": APPSPOT_APP_INIT,
        "params": APPSPOT_PLATFORM_PARAMS
    },

    # /arena/ #
    {
        "path": "/arena/getRecords",
        "params": {
            "type": "alltime"
        }
    },
    {
        "path": "/arena/getPersonalBest",
        "params": {}
    },

    # /char/ #
    {
        "path": "/char/list",
        "params": {
            "do_login": True
        }
    },

    # /credits/ #
    {
        "path": "/credits/token",
        "params": {
            "type": "Unity"
        }
    },
    {
        "path": "/credits/getoffers",
        "params": {}
    },

    # /dailyLogin/ #
    {
        "path": "/dailyLogin/fetchCalendar",
        "params": {}
    },

    # /friends/ #
    {
        "path": "/friends/getRequests",
        "params": {
            "targetName": ""
        },
        "path": "/friends/getList",
        "params": {}
    },

    # /fame/ #
    {
        "path": "/fame/list",
        "params": {
            "timespan": "all",
            # charId
            # accountId
        }
    },

    # /inGameNews/ #
    {
        "path": "/inGameNews/getNews",
        "params": {}
    },

    # /mysterybox/ #
    {
        "path": "/mysterybox/getBoxes",
        "params": {
            "language": "en",
            "version": "0",  
        }
    },

    # /package/ #
    {
        "path": "/package/getPackages",
        "params": {}
    },

    # /supportCampaign/ #
    {
        "path": "/supportCampaign/claim",
        "params": {}
    },
    {
        "path": "/supportCampaign/unlock",
        "params": {}
    },
    {
        "path": "/supportCampaign/donate",
        "params": {}
    },
    {
        "path": "/supportCampaign/status",
        "params": {}
    },
    {
        "path": "/supportCampaign/create",
        "params": {}
    },
    {
        "path": "/supportCampaign/getinfo",
        "params": {}
    },
    {
        "path": "/supportCampaign/getUnitySupporters",
        "params": {
            "page": 1
        }
    },

    # /season/
    {
        "path": "/season/getSeasons",
        "params": {}
    },

    # /unityNews/
    {
        "path": "/unityNews/getNews",
        "params": {}
    },
]


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

DIFF2HTML_CLI = BINARIES_DIR / "diff2html-cli" / "bin" / "diff2html"