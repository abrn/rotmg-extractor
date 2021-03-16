#!/bin/bash
FILE=${1:-src/main.py}
cd "${0%/*}" # change directoy to the file's
nodemon --exec venv/Scripts/python.exe $FILE --ext py