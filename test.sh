#!/bin/bash

currentTag=$(git fetch origin | git tag -l | grep $TAG);
echo 'asd '"$currentTag"' asd '"$TAG";
if [[ -z "$currentTag" ]]; then
  # git tag $TAG;
  # git push origin $TAG;
  echo 'Version '"$TAG"' published'
else
  echo "Nothing to publish";
fi