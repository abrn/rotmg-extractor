import io
import logging
import sys
from datetime import datetime
from pathlib import Path


class Logger:
    """ Create a new logger interface
    Args: 'logger_type': string 
        the name of the logger or None
    """
    def __init__(self, logger_type: str = None):
        # create a new logger based on the type
        # if None, return the 'root' logger
        self.logger = logging.getLogger(logger_type)
        self.initialized = False

    def setup(self) -> None:
        """ Create a new custom logger interface with handlers and formatting options """
        if self.initialized:
            # todo: debug log here for calling setup() while already initialized
            return
        # Add the indent and loglevel formatters plus set the output format
        self.logger.addFilter(IndentFilter())
        self.logger.addFilter(LevelFilter())

        # Construct the final log message with a custom string
        self.formatter = logging.Formatter(
            fmt="%(indent_level)s%(opt_level)s%(message)s",
            datefmt="%Y-%m-%d %I:%M:%S %p"
        )

        # The default logging level is 'production', with INFO, ERROR and FATAL messages
        self.logger.setLevel(logging.INFO)
        # The logging level can be changed to 'development mode' showing DEBUG and TRACE messages
        #todo: make this the default loglevel in production only
        self.setup_formatter()
        self.initialized = True

    def setup_formatter(self) -> None:
        """ Add a formatter to the current logger class """

        #todo: add webhook notications for Discord and web status pages

        # Add a conslose logger with the system standard output
        console_out = logging.StreamHandler(sys.stdout)

        console_out.setFormatter(self.formatter)
        self.logger.addHandler(console_out)

    def set_file_log(self, file_path: Path, clear_formatters: bool = True):
        """ Set the current log file path
        Args:
            file_path (Path):
                a PathLike object to the new log file
            clear_formatters (bool) (optional, default=True)
                Remove all custom Formatters from the Logger class
        """
        if clear_formatters:
            self.logger.handlers = []
            #todo: 
            self.setup_formatter()
        file_path.parent.mkdir(parents=True, exist_ok=True)

        # Overwriting the last logfile with the 'w' flag to clear for the new one
        filelog = logging.FileHandler(filename=file_path, mode="w")
        filelog.setFormatter(self.formatter)
        self.logger.addHandler(filelog)

    def log(self, msg: str, level: int = logging.INFO) -> None:
        return self.logger.log(level, msg)

    def print_time(self) -> None:
        time = datetime.now().astimezone().strftime("%m-%d-%y %H:%M:%S")
        self.log(logging.INFO, str(time))

    def pipe(self, pipe: io.BufferedReader) -> None:
        for line in iter(pipe.readline, b""):
            line = line.decode().replace("\r", "").replace("\n", "")
            self.log(logging.INFO, line)


class IndentFilter(logging.Filter):
    spaces = 4  # todo: fix this ugly ass system making lines everywhere
    level = 0

    def filter(self, record):
        record.indent_level = " " * (IndentFilter.level * IndentFilter.spaces)
        return True


class LevelFilter(logging.Filter):
    min_level = logging.WARNING

    def filter(self, record):
        if record.levelno >= LevelFilter.min_level:
            record.opt_level = f'{record.levelname}: '
        else:
            record.opt_level = ''
        return True


logger = Logger()
