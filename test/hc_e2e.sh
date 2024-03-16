#!/bin/bash
#This helper script will run E2E UI tests for Hub Central (https://github.com/marklogic/marklogic-data-hub/tree/develop/marklogic-data-hub-central/ui/e2e)

#override java and node version for Jenkins
[[ $USER = 'builder' ]] && { export PATH=/home/builder/java/jdk1.8.0_201/bin:/home/builder/nodeJs/node-v14.15.4-linux-x64/bin/:$PATH; unset JAVA_HOME; }

echo "---- start port forwarding ----"
kubectl port-forward hc-0 8000 8001 8002 8010 8011 8013 &> /dev/null &
forwarderPID=$!

echo "---- configure environment ----"
cd marklogic-data-hub/
rm -rf .gradle/
mkdir .gradle
export GRADLE_USER_HOME=$PWD/.gradle
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
npm run cy:run --reporter junit --reporter-options "toConsole=false"

echo "---- cleanup resources ----"
kill $bootRunPID $forwarderPID
rm -rf ${GRADLE_USER_HOME}/*
rm -rf ${GRADLE_USER_HOME}/ || ( ls -a ${GRADLE_USER_HOME} && exit 1 )
