# Build packer image in current directory.
# $1 - pass `1` to skip image existence check.
function build_image()
{
  export SCRIPT_PATH="$(pwd)"
  source ../../../common/build/include.sh

  if [ "$1" != "1" ]; then
    if image_exists; then
      >&2 echo "Image already exists in cloud, skipping build."
      return 0
    fi
  else
    >&2 echo "Force rebuild image."
  fi

  if [ -z "$SKIP_VAULT_BUILD" ]; then
    >&2 echo "Building vault binary"
    pushd ../../../common/vault
    ./build_vault.sh
    code=$?
    popd
    if [ $code -eq 0 ]; then
      >&2 echo "Build vault binary succeed."
    else
      >&2 echo "Build vault binary error occurred."
      return 1
    fi
  fi

  packer build \
    -var-file=../../../variables.pkrvars.hcl \
    -var 'image_sources_checksum='"$(image_sources_checksum)"'' \
    -force \
    build.pkr.hcl

  return $?
}

# Build packer images in all directories in the current directory.
# To target image build from a particular directory you can pass `<directoryName>` as a first parameter.
# Also you can pass the `--force` parameter to prevent checksum check and force image rebuilding.
function build_images()
{
  for var in "$@"; do
    if [ "$var" == "--force" ]; then
      force="1"
      continue
    fi
    target="$var"
  done

  for d in */ ; do
    folder="$(basename -- "$d")"
    if [ -n "$target" ] && [ "$target" != "$folder" ]; then
      >&2 echo "Skipping image build from directory '$folder' because target is '$target'"
      continue
    fi
    if [ -z "$target" ] && [ -n "$SKIP_DIRECTORY" ] && [ "$SKIP_DIRECTORY" == "$folder" ]; then
      continue
    fi

    >&2 echo "Building image from directory '$folder'"
    pushd "$d"
    if [ -f "./build.sh" ]; then
      echo "Custom build.sh found - using it."
      ./build.sh "$force"
      code=$?
    else
      build_image "$force"
      code=$?
    fi
    popd

    if [ "$code" == "0" ]; then
      >&2 echo "Build image succeed."
    else
      >&2 echo "Build image error occurred."
      exit 1
    fi
  done
}
