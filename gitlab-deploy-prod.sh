#!/bin/bash

#Get servers list
set -f
string=$DEPLOY_SERVER

# shellcheck disable=SC2206
array=(${string//,/ })

#Iterate servers for deploy and pull new version
for i in "${!array[@]}"; do
  echo "Deploy project on server" "${array[i]}"
  ssh ec2-user@"${array[i]}" ". update.sh"
done
