#!/usr/bin/env bash

pip install virtualenv
virtualenv scripts/environment
source scripts/environment/bin/activate
pip install -r scripts/requirements.txt
python scripts/unseal.py

deactivate
