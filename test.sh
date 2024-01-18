CURRENT_TAG=$(git fetch origin | git tag -l | grep $TAG);
echo 'asd '"$CURRENT_TAG"' asd '"$TAG";
if [[ -z "$CURRENT_TAG" ]]; then
  # git tag $TAG;
  # git push origin $TAG;
  echo 'Version '"$TAG"' published'
else
  echo "Nothing to publish";
fi