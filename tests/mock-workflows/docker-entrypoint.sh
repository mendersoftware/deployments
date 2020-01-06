#!/bin/sh

cd /app
pip install -r requirements.txt
python -m flask run -h 0.0.0.0 -p 8080
