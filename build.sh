#!/bin/bash -x

#go get -f -u github.com/golang/protobuf/proto
#go get -f -u github.com/golang/protobuf/protoc-gen-go
#go install github.com/golang/protobuf/protoc-gen-go

#go get -f -u github.com/gogo/protobuf/protobuf
#go install github.com/gogo/protobuf/protobuf

$GOPATH/bin/protoc \
    -I=. -I=$GOPATH/src -I=$GOPATH/src/github.com/gogo/protobuf/protobuf \
    --gogoslick_out=$GOPATH/src/github.com/demskie/hyperionfive/gamestate/ \
    --proto_path=$GOPATH/src/github.com/demskie/hyperionfive/gamestate/ gamestate.proto

#$GOPATH/bin/protoc \
#    --go_out=$GOPATH/src/github.com/demskie/hyperionfive/gamestate/ \
#    --proto_path=$GOPATH/src/github.com/demskie/hyperionfive/gamestate/ gamestate.proto

$GOPATH/bin/protoc \
    --plugin="protoc-gen-ts=/usr/local/bin/protoc-gen-ts" \
	--ts_out=$HOME/OneDrive/react-hyperion5-game/src/components/ \
	--js_out=import_style=commonjs,binary:$HOME/OneDrive/react-hyperion5-game/src/components/ \
    --proto_path=$GOPATH/src/github.com/demskie/hyperionfive/gamestate/ gamestate.proto

echo '/* eslint-disable */' | cat - $HOME/OneDrive/react-hyperion5-game/src/components/gamestate_pb.js  > temp && mv temp $HOME/OneDrive/react-hyperion5-game/src/components/gamestate_pb.js 