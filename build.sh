set -euxo pipefail

GOOS=linux
GOARCH=amd64
GO111MODULE=on

GOBIN=$(PWD)/functions go get ./...

if [[ -z ./functions ]]; then
  # create output directory
  mkdir -p functions;
fi

for v in $(pwd)/cmd/*; do
  for n in $v/*; do
    # strip trailing slash
    v=${v%*/}
    n=${n%*/}

    # extract directory name
    v=${v##*/}
    n=${n##*/}

    # build binary
    go build -o $(pwd)/functions/${v}_${n} $(pwd)/cmd/$v/$n;
  done
done