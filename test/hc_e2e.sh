#!/bin/bash
#This helper script will run E2E UI tests for Hub Central (https://github.com/marklogic/marklogic-data-hub/tree/develop/marklogic-data-hub-central/ui/e2e)

echo "---- start port forwarding ----"
kubectl port-forward hc-marklogic-0 8000 8001 8002 8010 8011 8013 &
forwarderPID=$!

echo "---- configure environment ----"
cd marklogic-data-hub/
./gradlew clean build -x test
./gradlew publishToMavenLocal -PskipWeb=true
cd marklogic-data-hub-central/ui/e2e
./setup.sh dhs=false mlHost=localhost mlSecurityUsername=admin mlSecurityPassword=admin

echo "---- start Hub Central application ----"
cd ../../..
./gradlew bootRun -PhubUseLocalDefaults=true &
bootRunPID=$!

sleep 10

echo "---- start UI sanity tests on HC ----"
cd marklogic-data-hub-central/ui/e2e
#npm run cy:run --reporter junit --reporter-options "toConsole=false"
npm run cy:run-sanity --reporter junit --reporter-options "toConsole=false"

echo "---- cleanup background processes ----"
kill $bootRunPID
kill $forwarderPID
