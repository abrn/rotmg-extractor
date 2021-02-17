import logging
import sys
from datetime import datetime
from pathlib import Path

from classes import Constants


class Logger:

    def __init__(self):
        self.logger = logging.getLogger()

    def setup(self):
        self.logger.addFilter(IndentFilter())
        self.logger.addFilter(LevelFilter())

        # formatter = logging.Formatter(
        #     fmt="%(asctime)s %(levelname)-8s %(indent_level)s %(message)s",
        #     datefmt="%Y-%m-%d %I:%M:%S %p"
        # )

        formatter = logging.Formatter(
            fmt="%(indent_level)s%(opt_level)s%(message)s",
            datefmt="%Y-%m-%d %I:%M:%S %p"
        )

        # Log to console
        syslog = logging.StreamHandler(sys.stdout)
        syslog.setFormatter(formatter)
        self.logger.addHandler(syslog)

        # Log to file ./temp/current/log.txt
        Path(Constants.WORK_DIR).mkdir(parents=True, exist_ok=True)
        filelog = logging.FileHandler(
            filename=Constants.WORK_DIR / "log.txt",
            mode="w"  # clear log first
        )
        filelog.setFormatter(formatter)
        self.logger.addHandler(filelog)

        # self.logger.setLevel(logging.DEBUG)
        self.logger.setLevel(logging.INFO)
        self.log(logging.INFO, str(datetime.now().astimezone().strftime("%Y-%m-%dT%H:%M:%S %z %Z")))
        # self.log(logging.INFO, current_time_iso8601())

    def log(self, level, msg):
        return self.logger.log(level, msg)


class IndentFilter(logging.Filter):
    spaces = 4
    level = 0

    def filter(self, record):
        record.indent_level = " " * (IndentFilter.level * IndentFilter.spaces)
        return True


class LevelFilter(logging.Filter):
    min_level = logging.WARNING
    def filter(self, record):

        if record.levelno >= LevelFilter.min_level:
            record.opt_level = record.levelname + ": "
        else:
            record.opt_level = ""

        return True 

logger = Logger()
