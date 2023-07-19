#! /bin/bash

OLD_PWD=`pwd`
WORK_DIR=`dirname "$0"`
cd "$WORK_DIR"

make clean

cd "$OLD_PWD"

