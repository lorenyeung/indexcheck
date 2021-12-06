#!/bin/bash
GIT_REPO=$(jq -r '.git_repo' metadata.json)
BINARY_PREFIX=$(jq -r '.binary_prefix' metadata.json)
LATEST_SCRIPT_TAG=$(curl -s https://api.github.com/repos/lorenyeung/${GIT_REPO}/releases/latest | jq -r '.tag_name')
test=$(curl -s https://api.github.com/repos/lorenyeung/${GIT_REPO}/releases/latest)
echo "test var $test"
LOCAL_TAG_NAME=$(jq -r '.script_version' metadata.json)
LOCAL_NAME=$(echo "${LOCAL_TAG_NAME/v/}")
RELEASE_BRANCH=$(jq -r '.release_branch' metadata.json)

if [ "$LATEST_SCRIPT_TAG" = "$LOCAL_TAG_NAME" ]; then
    echo "Did you forget to increment metadata.json? The latest release on github ($LATEST_SCRIPT_TAG) is the same as metadata.json's ($LOCAL_TAG_NAME)"
    select yn in "Yes" "No"; do
        case $yn in
            Yes)
                echo "Version please (no v):"
                read LOCAL_NAME
                LOCAL_TAG_NAME=v$LOCAL_NAME
                file=$(jq -r '.script_version="'$LOCAL_TAG_NAME'"' metadata.json)
                echo $file > metadata.json
                echo "double check:"
                cat metadata.json
                break;;
            No) echo "OK" ; break;;
        esac
    done
    else
        echo "Latest Github Tag:$LATEST_SCRIPT_TAG Local Tag:$LOCAL_TAG_NAME Local Name:$LOCAL_NAME"
fi
echo "Enter body description"
read message
body="{
  \"tag_name\": \"$LOCAL_TAG_NAME\",
  \"target_commitish\": \"$RELEASE_BRANCH\",
  \"name\": \"$LOCAL_NAME\",
  \"body\": \"$message\",
  \"draft\": false,
  \"prerelease\": false
}"
object=$(git rev-parse HEAD)
echo "last commit is $object"
tag_body="{
  \"tag\": \"$LOCAL_TAG_NAME\",
  \"message\": \"$message\",
  \"object\": \"$object\",
  \"type\": \"commit\"
}"
echo "Body:"
echo "$body"
#echo "Tag Body:"
#echo $tag_body
echo "Looks good?"
    select yn in "Yes" "No"; do
        case $yn in
            Yes)
                echo $body > release.json 
                curl -u $GIT_USER:$GIT_TOKEN -XPOST "https://api.github.com/repos/lorenyeung/$GIT_REPO/releases" -H "Content-Type: application/json" -T release.json -o release-response.json
                rm release.json
                make build
                BINARIES=("$BINARY_PREFIX-darwin-x64" "$BINARY_PREFIX-linux-x64")
                ASSET_URL=$(jq -r '.upload_url' release-response.json)
                edited=$(echo $ASSET_URL | sed 's/{?name,label}//')

                for BINARY in ${BINARIES[@]}; do
                    URL="$edited?name=$BINARY&label=$BINARY"
                    echo "uploading $BINARY to $URL"
                    curl -T $BINARY "$URL" -H "Content-type: application/x-binary" -u $GIT_USER:$GIT_TOKEN
                done
                echo $edited
                rm release-response.json
                rm $BINARY_PREFIX-*
                rm $TEMPLATE
                break;;
            No) echo "OK" ; break;;
        esac
    done

