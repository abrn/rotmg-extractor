import urllib
import xmltodict
from .Constants import APP_INIT_PATH


class AppSettings:
    def __init__(self, url):
        self.url = url
        self.__get()

    def __get(self):
        url = self.url + APP_INIT_PATH
        self.xml = urllib.request.urlopen(url).read()
        data = xmltodict.parse(self.xml)

        # <BuildId>rotmg-exalt-win-64</BuildId>
        # <BuildHash>a1c8d9ae2a2508dcc3994b33dd6a803a</BuildHash>
        # <BuildVersion>a9cb33d6944a7f8bbf7181c71cc932f11ed85ba3</BuildVersion>
        # <BuildCDN>https://rotmg-build.decagames.com/build-release/</BuildCDN>
        self.client = {}
        self.client["build_id"] = data["AppSettings"].get("BuildId")
        self.client["build_hash"] = data["AppSettings"].get("BuildHash")
        self.client["build_version"] = data["AppSettings"].get("BuildVersion")
        self.client["build_cdn"] = data["AppSettings"].get("BuildCDN")

        # <LauncherBuildId>RotMG-Exalt-Installer</LauncherBuildId>
        # <LauncherBuildHash>d554e291899750f9d36c750798e85646</LauncherBuildHash>
        # <LauncherBuildVersion>386777c109b1f7385ab1636bc7e82a1d7f451352</LauncherBuildVersion>
        # <LauncherBuildCDN>https://rotmg-build.decagames.com/launcher-release/</LauncherBuildCDN>
        self.launcher = {}
        self.launcher["build_id"] = data["AppSettings"].get("LauncherBuildId")
        self.launcher["build_hash"] = data["AppSettings"].get("LauncherBuildHash")
        self.launcher["build_version"] = data["AppSettings"].get("LauncherBuildVersion")
        self.launcher["build_cdn"] = data["AppSettings"].get("LauncherBuildCDN")
