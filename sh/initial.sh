#!/bin/bash

cp .env.example .env
cp config.yaml.example config.yaml

set -a
source .env
set +a

make b
