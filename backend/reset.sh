#!/bin/bash
echo "Restarting Ritme project ..."

/usr/bin/php /var/www/ritme/artisan optimize:clear

supervisorctl stop ritme-queue
supervisorctl stop ritme-scheduler

supervisorctl start ritme-queue
supervisorctl start ritme-scheduler

echo "Done."
