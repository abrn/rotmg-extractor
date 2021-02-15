import logging
import os
import subprocess

from classes import Constants, logger
from classes.CustomLogger import IndentFilter


def unpack_launcher_assets(launcher_path, output_path):

    unpacker_file = None
    if os.name == "nt":
        unpacker_file = Constants.LAUNCHER_UNPACKER_WINDOWS
    elif os.name == "posix":
        unpacker_file = Constants.LAUNCHER_UNPACKER_LINUX
    else:
        return

    logger.log(logging.INFO, "Unpacking launcher assets...")
    IndentFilter.level += 1

    subprocess.run(
        [unpacker_file, launcher_path, output_path],
        stdout=subprocess.DEVNULL,
    )

    logger.log(logging.INFO, "Done!")
    IndentFilter.level -= 1
