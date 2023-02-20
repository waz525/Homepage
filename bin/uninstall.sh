#!/bin/sh

echo "Remove configure from supervisor ..."
rm -f /etc/supervisord.d/Waiters.ini
supervisorctl update

echo "Kill all Waiters ..."
killall -9 Waiters

echo "Remove Waiters ..."
rm -rf /home/Waiters

echo "Waiters Uninstall Success !"


