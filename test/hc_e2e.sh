#!/bin/bash
#This helper script will run E2E UI tests for Hub Central (https://github.com/marklogic/marklogic-data-hub/tree/develop/marklogic-data-hub-central/ui/e2e)

#override java and node version for Jenkins
[[ $USER = 'builder' ]] && { export PATH=/home/builder/java/jdk1.8.0_201/bin:/home/builder/nodeJs/node-v16.19.1-linux-x64/bin/:$PATH; unset JAVA_HOME; }

echo "---- start port forwarding ----"
kubectl port-forward hc-marklogic-0 8000 8001 8002 8010 8011 8013 &> /dev/null &
forwarderPID=$!

echo "---- configure environment ----"
cd marklogic-data-hub/
./gradlew -Dhttps.protocols=TLSv1.2 clean build -x test
./gradlew -Dhttps.protocols=TLSv1.2 publishToMavenLocal -PskipWeb=true
cd marklogic-data-hub-central/ui/e2e
./setup.sh dhs=false mlHost=localhost mlSecurityUsername=admin mlSecurityPassword=admin

echo "---- start Hub Central application ----"
cd ../../..
./gradlew -Dhttps.protocols=TLSv1.2 bootRun -PhubUseLocalDefaults=true &
bootRunPID=$!

sleep 10

echo "---- start UI sanity tests on HC ----"
cd marklogic-data-hub-central/ui/e2e
npm run cy:run --reporter junit --reporter-options "toConsole=false"

echo "---- cleanup background processes ----"
kill $bootRunPID $forwarderPID
