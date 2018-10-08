go get -f -u -v github.com\golang\protobuf\proto
go get -f -u -v github.com\golang\protobuf\protoc-gen-go
go install github.com\golang\protobuf\protoc-gen-go

go get -f -u github.com\gogo\protobuf\protoc-gen-gogoslick
go install github.com\gogo\protobuf\protoc-gen-gogoslick

CMD /C npm install -g ts-protoc-gen

CMD /C npm install -g tslint typescript