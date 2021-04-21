import io
import logging
import sys
from datetime import datetime
from pathlib import Path

from classes import Constants


class Logger:

    def __init__(self):
        self.logger = logging.getLogger()
        self.initialized = False

    def setup(self):
        if self.initialized:
            return

        self.logger.addFilter(IndentFilter())
        self.logger.addFilter(LevelFilter())

        self.formatter = logging.Formatter(
            fmt="%(indent_level)s%(opt_level)s%(message)s",
            datefmt="%Y-%m-%d %I:%M:%S %p"
        )

        self.setupHandlers()

        self.logger.setLevel(logging.INFO)
        self.initialized = True

    def setupHandlers(self):
        # Log to console
        syslog = logging.StreamHandler(sys.stdout)
        syslog.setFormatter(self.formatter)
        self.logger.addHandler(syslog)

    def setFileLog(self, file_path: Path, clearHandlers=True):
        # Used to clear old file logs
        if clearHandlers:
            self.logger.handlers = []
            self.setupHandlers()

        file_path.parent.mkdir(parents=True, exist_ok=True)
        filelog = logging.FileHandler(
            filename=file_path,
            mode="w"  # clear log first
        )
        filelog.setFormatter(self.formatter)
        self.logger.addHandler(filelog)

    def log(self, level, msg):
        return self.logger.log(level, msg)

    def printTime(self):
        self.log(logging.INFO, str(datetime.now().astimezone().strftime("%Y-%m-%dT%H:%M:%S %z %Z")))
        
    def pipe(self, pipe: io.BufferedReader):
        for line in iter(pipe.readline, b""):
            line = line.decode().replace("\r\n", "")
            self.log(logging.INFO, line)


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
