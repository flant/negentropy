#!/usr/bin/env bash

pip install virtualenv
virtualenv scripts/e2e
source scripts/e2e/bin/activate
pip install -r scripts/requirements.txt
python scripts/unseal.py

deactivate