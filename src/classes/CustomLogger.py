import logging
import sys

from classes import Constants


class Logger:

    def __init__(self):
        self.logger = logging.getLogger()

    def setup(self):
        self.logger.addFilter(IndentFilter())

        formatter = logging.Formatter(
            fmt="%(asctime)s %(levelname)-8s %(indent_level)s %(message)s",
            datefmt="%Y-%m-%d %I:%M:%S %p"
        )

        # Log to console
        syslog = logging.StreamHandler(sys.stdout)
        syslog.setFormatter(formatter)
        self.logger.addHandler(syslog)

        # Log to file ./temp/current/log.txt
        filelog = logging.FileHandler(
            filename=Constants.WORK_DIR / "log.txt",
            mode="w"  # clear log first
        )
        filelog.setFormatter(formatter)
        self.logger.addHandler(filelog)

        # self.logger.setLevel(logging.NOTSET)
        # self.logger.setLevel(logging.DEBUG)
        self.logger.setLevel(logging.INFO)

    def log(self, level, msg):
        return self.logger.log(level, msg)


class IndentFilter(logging.Filter):
    spaces = 4
    level = 0

    def filter(self, record):
        record.indent_level = " " * (IndentFilter.level * IndentFilter.spaces)
        return True


logger = Logger()
