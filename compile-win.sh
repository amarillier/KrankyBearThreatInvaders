#!/usr/bin/env bash

set -euo pipefail

echo "KrankyBear ThreatInvaders - Windows Sync, Compile. Package and Sync Back (bash)"
echo "================================================"
echo

# Create bin directory if it doesn't exist
if [ ! -d "bin" ]
then
    mkdir -p bin
fi

# Cleanup previous Windows binary only (keep other artifacts)
rm -f threatinvaders-windows.exe
rm -f bin/*
rm -f installers/*

echo "Windows"
echo "syncing to windows"
./sync2windows.sh

echo "compiling on windows"
./compile-windows-ssh.sh -windows -package

echo "syncing back"
./sync2windows.sh sync-back
rm -f threatinvaders-windows.exe

echo "Not setting icons"
# ./setIcon.sh Resources/Images/KrankyBearUSArmyCombatHelmetWoodland.png bin/threatinvaders-windows.exe

# "Now this is not even the end. It is not even the beginning of the end. But it is, perhaps, the end of the beginning." Winston Churchill, November 10, 1942
