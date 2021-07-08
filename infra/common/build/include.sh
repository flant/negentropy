# Calculates cumulative checksum for all used in image's configuration file common scripts.
# $1 - image path
function used_common_scripts_checksum() {
  target_path="$1"

  common_scripts_path="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
  common_scripts_path=$(dirname "$common_scripts_path")
  common_scripts_path="$common_scripts_path/packer-scripts/"

  checksums=()
  for common_script_path in $(cat $target_path/build.pkr.hcl | grep "packer-scripts" | tr -d '", '); do
    common_script=${common_script_path##*/}
    common_script_checksum="$( cd "$common_scripts_path"; git ls-tree @ -- "$common_script" | awk '{print $3}' )"
    checksums+=("$common_script_checksum")
  done

  IFS=$'\n' sorted_checksums=($(sort <<<"${checksums[*]}")); unset IFS
  joined_checksums="$(printf "%s" "${sorted_checksums[@]}")"
  echo $(echo -n "$joined_checksums" | sha1sum | awk '{print $1}')
}

# Calculates cumulative checksum for base image.
function base_image_sources_checksum()
{
  path="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
  path=$(dirname "$path")
  path="$path/packer/01-alpine-base"
  dir_name=${path##*/}
  checksums=("$( cd "$path"; cd ..; git ls-tree @ -- "$dir_name" | awk '{print $3}' )")
  checksums+=("$(used_common_scripts_checksum "$path")")
  IFS=$'\n' sorted_checksums=($(sort <<<"${checksums[*]}")); unset IFS
  joined_checksums="$(printf "%s" "${sorted_checksums[@]}")"
  echo $(echo -n "$joined_checksums" | sha1sum | awk '{print $1}')
}

# Outputs checksum for `vault-plugins` directory.
function vault_plugins_checksum()
{
  path="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
  path=$(dirname "$path")
  path=$(dirname "$path")
  path=$(dirname "$path")
  path="$path/vault-plugins"
  dir_name=${path##*/}
  echo "$( cd "$path"; cd ..; git ls-tree @ -- "$dir_name" | awk '{print $3}' )"
}

# Calculates cumulative checksum for current image.
function image_sources_checksum()
{
  dir_name=${SCRIPT_PATH##*/}
  checksums=("$( cd "$SCRIPT_PATH"; cd ..; git ls-tree @ -- "$dir_name" | awk '{print $3}' )")
  # If we building outside of common directory we need to add base image sources checksum
  # to rebuilt image if base image changed.
  if [[ "$SCRIPT_PATH" != *"infra/common/packer"* ]]; then
    checksums+=("$(base_image_sources_checksum)")
  fi
  # Add summarised checksum of all used common-scripts.
  checksums+=("$(used_common_scripts_checksum "$SCRIPT_PATH")")
  # If we building vault image we need to add vault-plugins checksum.
  if [[ "$SCRIPT_PATH" == *"vault"* ]]; then
    checksums+=("$(vault_plugins_checksum)")
  fi
  # Sort checksums to avoid possible flapping.
  IFS=$'\n' sorted_checksums=($(sort <<<"${checksums[*]}")); unset IFS
  # Join all checksums and output single checksum of all checksums.
  joined_checksums="$(printf "%s" "${sorted_checksums[@]}")"
  echo $(echo -n "$joined_checksums" | sha1sum | awk '{print $1}' | head -c8)
}

# Outputs resulting image name for current image.
function image_name()
{
    echo "$(packer inspect -var-file="$SCRIPT_PATH"/../../../variables.pkrvars.hcl -var 'image_sources_checksum='"$(image_sources_checksum)"'' "$SCRIPT_PATH"/build.pkr.hcl | grep local.image_name | awk -F' ' '{ print $2 }' | tr -d '"')"
}

# Checks if resulting image already exists in the cloud.
function image_exists()
{
  if gcloud compute images list --filter="name=('"$(image_name)"')" --format=json | jq -e 'length > 0' > /dev/null; then
    return 0
  fi
  return 1
}
