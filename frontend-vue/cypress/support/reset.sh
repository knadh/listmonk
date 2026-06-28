#!/bin/bash

pkill -9 listmonk
 cd ../
./listmonk --install --yes
./listmonk > /dev/null 2>/dev/null &
