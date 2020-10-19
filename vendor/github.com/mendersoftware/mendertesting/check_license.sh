#!/bin/sh

set -e

ret=0

################################################################################
# Check main license file.
################################################################################

# Find commit with latest author date (which is surprisingly difficult!).
LATEST=`git log --no-merges --format="%at %H" | sort -rn | head -n1 | cut -d' ' -f2`
LATEST_YEAR=`git log -n1 --format=%ad --date=format:%Y $LATEST`

if ! grep -iq "Copyright *$LATEST_YEAR *Northern.tech" LICENSE; then
    echo "'Copyright $LATEST_YEAR Northern.tech' not found in LICENSE. Wrong year maybe?"
    ret=1
fi

################################################################################
# Check license of dependencies.
################################################################################

CHKSUM_FILE=LIC_FILES_CHKSUM.sha256

while [ -n "$1" ]; do
    case "$1" in
        --add-license=*)
            file="${1#--add-license=}"
            KNOWN_LICENSE_FILES="$KNOWN_LICENSE_FILES $file"
            # The file must exist in LIC_FILES_CHKSUM.sha256
            if ! fgrep -q "$file" $CHKSUM_FILE
            then
                echo "$file does not have a checksum in $CHKSUM_FILE"
                exit 1
            fi
            ;;
        -*)
            echo "Usage: $(basename "$0") <dir-to-check>"
            exit 1
            ;;
    esac
    shift
done

if [ -n "$1" ]
then
    cd "$1"
fi

# Remove all newlines from the Checksum file as these are reported as formatting
# errors by the shasum program
TMP_CHKSUM_FILE=$(mktemp)
trap cleanup EXIT
cleanup()
{
  rm $TMP_CHKSUM_FILE
}
sed '/^$/d' $CHKSUM_FILE > $TMP_CHKSUM_FILE

# Use the tmp-file for the rest of the script
CHKSUM_FILE=$TMP_CHKSUM_FILE

# Collect only stderr from the subcommand
output="$(exec 3>&1; shasum --warn --algorithm 256 --check $CHKSUM_FILE >/dev/null 2>&3)"

if echo "$output" | grep -q 'line is improperly formatted' -; then
  echo >&2 "Some line(s) in the LIC_FILE_CHKSUM.sha256 file are misformed"
  cat $CHKSUM_FILE
  exit 1
fi

# Unlisted licenses not allowed.
for file in $(find . -iname 'LICEN[SC]E' -o -iname 'LICEN[SC]E.*' -o -iname 'COPYING')
do
    file=$(echo $file | sed -e 's,./,,')
    if ! fgrep "$(shasum -a 256 $file)" $CHKSUM_FILE > /dev/null
    then
        echo "$file has missing or wrong entry in $CHKSUM_FILE"
        ret=1
    fi
done

# There must be a license at the top level.
if [ LICENSE* = "LICENSE*" ] && [ COPYING* = "COPYING*" ]
then
    echo "No top level license file."
    ret=1
fi

# There must be a license at the top level of each Go dependency.
# The logic is so that each .go source file must have a license file in the same
# directory, or in a parent directory.
for dep_dir in Godeps/_workspace/src vendor
do
    if [ -d "$dep_dir" ]
    then
        for gofile in $(find "$dep_dir" -name '*.go')
        do
            parent_dir="$(dirname "$gofile")"
            found=0
            while [ "$parent_dir" != "$dep_dir" ]
            do
                # Either we need to find a license file, or the file must be
                # covered by one of the license files specified in
                # KNOWN_LICENSE_FILES.
                if [ $(find "$parent_dir" -maxdepth 1 -iname 'LICEN[SC]E' -o -iname 'LICEN[SC]E.*' -o -iname 'COPYING' | wc -l) -ge 1 ]
                then
                    found=1
                    break
                fi
                if [ -n "$KNOWN_LICENSE_FILES" ]
                then
                    for known_file in $KNOWN_LICENSE_FILES
                    do
                        if [ "$(dirname $known_file)" = "$parent_dir" ]
                        then
                            found=1
                            break 2
                        fi
                    done
                fi
                parent_dir="$(dirname "$parent_dir")"
            done
            if [ $found != 1 ]
            then
                echo "No license file to cover $gofile"
                ret=1
                break
            fi
        done
    fi
done

exit $ret
