#!/bin/bash

# Prepare the environment:
if [ ! -z "${WORKSPACE}" ]; then
  source jenkins/environment.sh
fi

export LANG="en_US.UTF-8"
export LC_ALL="en_US.UTF-8"

make
make test
