function git_directory_checksum()
{
  dir_name=${SCRIPT_PATH##*/}
  echo "$( cd "$SCRIPT_PATH"; cd ..; git ls-tree @ -- "$dir_name" | awk '{print $3}' )"
}

function image_name() {
    echo "$(packer inspect -var 'git_directory_checksum='"$(git_directory_checksum)"'' "$SCRIPT_PATH"/build.pkr.hcl | grep local.image_name | awk -F' ' '{ print $2 }' | tr -d '"')"
}

function image_exists()
{
  if gcloud compute images list --filter="name=('"$(image_name)"')" --format=json | jq -e 'length > 0' > /dev/null; then
    return 0
  fi
  return 1
}
