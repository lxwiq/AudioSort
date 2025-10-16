"""
Professional logging system with visual animations for AudioSort.
Clean, emoji-free output with progress indicators and animations.
"""

import sys
import time
import threading
from typing import Optional, TextIO
from enum import Enum


class LogLevel(Enum):
    DEBUG = "DEBUG"
    INFO = "INFO"
    SUCCESS = "SUCCESS"
    WARNING = "WARNING"
    ERROR = "ERROR"


class Color:
    """ANSI color codes for terminal output."""
    RESET = '\033[0m'
    BLACK = '\033[30m'
    RED = '\033[31m'
    GREEN = '\033[32m'
    YELLOW = '\033[33m'
    BLUE = '\033[34m'
    MAGENTA = '\033[35m'
    CYAN = '\033[36m'
    WHITE = '\033[37m'
    GRAY = '\033[90m'
    BOLD = '\033[1m'
    DIM = '\033[2m'


class Spinner:
    """Animated spinner for long operations."""

    def __init__(self, message: str, delay: float = 0.1):
        self.message = message
        self.delay = delay
        self.spinner_chars = ['⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏']
        self.running = False
        self.spinner_thread = None

    def start(self):
        """Start the spinner animation."""
        self.running = True
        self.spinner_thread = threading.Thread(target=self._spin)
        self.spinner_thread.daemon = True
        self.spinner_thread.start()

    def stop(self):
        """Stop the spinner animation."""
        self.running = False
        if self.spinner_thread:
            self.spinner_thread.join()
        # Clear the spinner line
        sys.stdout.write('\r' + ' ' * (len(self.message) + 10) + '\r')
        sys.stdout.flush()

    def _spin(self):
        """Internal spinning logic."""
        while self.running:
            for char in self.spinner_chars:
                if not self.running:
                    break
                sys.stdout.write(f'\r{Color.CYAN}{char}{Color.RESET} {self.message}')
                sys.stdout.flush()
                time.sleep(self.delay)


class ProgressBar:
    """Text-based progress bar for file operations."""

    def __init__(self, total: int, width: int = 50, prefix: str = "Progress"):
        self.total = total
        self.width = width
        self.prefix = prefix
        self.current = 0

    def update(self, increment: int = 1):
        """Update progress bar."""
        self.current += increment
        if self.current > self.total:
            self.current = self.total
        self._draw()

    def _draw(self):
        """Draw the progress bar."""
        percent = self.current / self.total
        filled_width = int(self.width * percent)
        bar = '█' * filled_width + '░' * (self.width - filled_width)

        sys.stdout.write(f'\r{Color.BLUE}{self.prefix}: {Color.RESET}|{Color.GREEN}{bar}{Color.RESET}| '
                        f'{Color.CYAN}{self.current}/{self.total}{Color.RESET} '
                        f'{Color.BOLD}{int(percent * 100)}%{Color.RESET}')
        sys.stdout.flush()

    def finish(self):
        """Complete the progress bar."""
        self.current = self.total
        self._draw()
        sys.stdout.write('\n')
        sys.stdout.flush()


class AudioSortLogger:
    """Professional logger with clean output and animations."""

    def __init__(self, name: str = "audiosort", level: LogLevel = LogLevel.INFO,
                 output: Optional[TextIO] = None, show_time: bool = True):
        self.name = name
        self.level = level
        self.output = output or sys.stdout
        self.show_time = show_time
        self.spinner: Optional[Spinner] = None
        self.progress_bar: Optional[ProgressBar] = None

        # Color mapping for different log levels
        self.colors = {
            LogLevel.DEBUG: Color.DIM + Color.GRAY,
            LogLevel.INFO: Color.WHITE,
            LogLevel.SUCCESS: Color.GREEN,
            LogLevel.WARNING: Color.YELLOW,
            LogLevel.ERROR: Color.RED + Color.BOLD
        }

        # Prefix mapping for different log levels
        self.prefixes = {
            LogLevel.DEBUG: "DEBUG",
            LogLevel.INFO: "INFO",
            LogLevel.SUCCESS: "SUCCESS",
            LogLevel.WARNING: "WARNING",
            LogLevel.ERROR: "ERROR"
        }

    def _should_log(self, level: LogLevel) -> bool:
        """Check if message should be logged based on current level."""
        level_order = {
            LogLevel.DEBUG: 0,
            LogLevel.INFO: 1,
            LogLevel.SUCCESS: 2,
            LogLevel.WARNING: 3,
            LogLevel.ERROR: 4
        }
        return level_order[level] >= level_order[self.level]

    def _format_message(self, level: LogLevel, message: str) -> str:
        """Format log message with timestamp and colors."""
        color = self.colors[level]
        prefix = self.prefixes[level]

        if self.show_time:
            timestamp = time.strftime("%H:%M:%S")
            return f"{Color.DIM}[{timestamp}]{Color.RESET} {color}[{prefix}]{Color.RESET} {color}{message}{Color.RESET}"
        else:
            return f"{color}[{prefix}]{Color.RESET} {color}{message}{Color.RESET}"

    def _write(self, level: LogLevel, message: str):
        """Write message to output."""
        if not self._should_log(level):
            return

        formatted = self._format_message(level, message)
        self.output.write(formatted + '\n')
        self.output.flush()

    def debug(self, message: str):
        """Log debug message."""
        self._write(LogLevel.DEBUG, message)

    def info(self, message: str):
        """Log info message."""
        self._write(LogLevel.INFO, message)

    def success(self, message: str):
        """Log success message."""
        self._write(LogLevel.SUCCESS, message)

    def warning(self, message: str):
        """Log warning message."""
        self._write(LogLevel.WARNING, message)

    def error(self, message: str):
        """Log error message."""
        self._write(LogLevel.ERROR, message)

    def start_spinner(self, message: str):
        """Start an animated spinner."""
        if self.spinner:
            self.spinner.stop()
        self.spinner = Spinner(message)
        self.spinner.start()

    def stop_spinner(self, success_message: Optional[str] = None):
        """Stop spinner and optionally show success message."""
        if self.spinner:
            self.spinner.stop()
            if success_message:
                self.success(success_message)
            self.spinner = None

    def start_progress(self, total: int, prefix: str = "Progress"):
        """Start a progress bar."""
        if self.progress_bar:
            self.progress_bar.finish()
        self.progress_bar = ProgressBar(total, prefix=prefix)

    def update_progress(self, increment: int = 1):
        """Update progress bar."""
        if self.progress_bar:
            self.progress_bar.update(increment)

    def finish_progress(self, message: Optional[str] = None):
        """Finish progress bar and optionally show completion message."""
        if self.progress_bar:
            self.progress_bar.finish()
            if message:
                self.success(message)
            self.progress_bar = None

    def header(self, title: str, width: int = 60):
        """Display a formatted header."""
        line = '=' * width
        padding = (width - len(title)) // 2

        self.info(line)
        self.info(' ' * padding + Color.BOLD + title + Color.RESET + ' ' * padding)
        self.info(line)

    def section(self, title: str):
        """Display a section header."""
        self.info(f"\n--- {Color.BOLD}{title}{Color.RESET} ---")

    def step(self, message: str, step_num: Optional[int] = None, total_steps: Optional[int] = None):
        """Display a step in a process."""
        if step_num and total_steps:
            prefix = f"[{step_num}/{total_steps}]"
            self.info(f"{Color.CYAN}{prefix}{Color.RESET} {message}")
        else:
            self.info(f"{Color.CYAN}{chr(187)}{Color.RESET} {message}")

    def file_operation(self, operation: str, source: str, destination: str):
        """Display file operation details."""
        self.info(f"{Color.BLUE}{operation}{Color.RESET}")
        self.info(f"  Source: {source}")
        self.info(f"  Destination: {destination}")

    def metadata_summary(self, metadata):
        """Display metadata summary."""
        self.info(f"{Color.BOLD}Metadata Summary:{Color.RESET}")
        if metadata.title:
            self.info(f"  Title: {metadata.title}")
        if metadata.authors:
            self.info(f"  Author(s): {', '.join(metadata.authors)}")
        if metadata.series:
            self.info(f"  Series: {metadata.series} #{metadata.series_position}")
        if metadata.duration_minutes:
            self.info(f"  Duration: {metadata.duration_minutes} minutes")
        if metadata.language:
            self.info(f"  Language: {metadata.language}")

    def operation_summary(self, successes: list, failures: list, skipped: list):
        """Display operation summary."""
        self.info("\n" + "=" * 50)

        if successes:
            self.success(f"Completed: {len(successes)} operations")
            for item in successes[:5]:  # Show first 5
                self.info(f"  {item}")
            if len(successes) > 5:
                self.info(f"  ... and {len(successes) - 5} more")

        if skipped:
            self.warning(f"Skipped: {len(skipped)} operations")
            for item in skipped[:3]:
                self.warning(f"  {item}")
            if len(skipped) > 3:
                self.warning(f"  ... and {len(skipped) - 3} more")

        if failures:
            self.error(f"Failed: {len(failures)} operations")
            for item in failures:
                self.error(f"  {item}")

        self.info("=" * 50)


# Global logger instance
logger = AudioSortLogger()