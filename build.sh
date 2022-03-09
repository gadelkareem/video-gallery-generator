#!/usr/bin/env bash

package_name="vgg"
echo "Building $package_name"

platforms=("windows/amd64" "windows/386" "windows/arm64" "darwin/amd64" "darwin/arm64" "linux/amd64" "linux/arm64" "pi/arm")

rm -rf build
mkdir -p build
for platform in "${platforms[@]}"
do
	platform_split=(${platform//\// })
	GOOS=${platform_split[0]}
	GOARCH=${platform_split[1]}
	output_name=$package_name'-'$GOOS'-'$GOARCH
	extra_ldflags=""
	if [ $GOOS = "windows" ]; then
		output_name+='.exe'
	fi

  echo "Building $output_name for $GOOS $GOARCH"
  if [ $GOOS = "pi" ]; then
    GOOS=linux
    extra_ldflags="CC_FOR_TARGET=arm-linux-gnueabi-gcc"
  fi
	env GOOS=$GOOS GOARCH=$GOARCH $extra_ldflags  go build -o build/$output_name $package
	if [ $? -ne 0 ]; then
   		echo 'An error has occurred! Aborting the script execution...'
		exit 1
	fi
	chmod +x build/$output_name
done

