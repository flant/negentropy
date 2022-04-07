#!/usr/bin/env python3

import os
import json
import argparse

from google.oauth2 import service_account

from apply import generate_packer_config


parser = argparse.ArgumentParser()
parser.add_argument('--generate-packer-config', dest='generate_packer_config')
args = parser.parse_args()

if args.generate_packer_config:
    generate_packer_config(args.generate_packer_config)
