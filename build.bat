SETLOCAL
SET JSOUTPATH=%OneDrive%\react-hyperion5-game\src\components\protobuf
SET GOPROJECTPATH=%GOPATH%\src\github.com\demskie\hyperionfive

%GOPATH%\bin\protoc ^
-I=. -I=%GOPATH%\src -I=%GOPATH%\src\github.com\gogo\protobuf\protobuf ^
--gogoslick_out=%GOPROJECTPATH%\backendmessage\ ^
--proto_path=%GOPROJECTPATH%\backendmessage\ backendmessage.proto &

%GOPATH%\bin\protoc ^
--js_out=import_style=commonjs,binary:%JSOUTPATH% ^
--proto_path=%GOPROJECTPATH%\backendmessage\ backendmessage.proto ^
--plugin="protoc-gen-ts=%APPDATA%\npm\protoc-gen-ts.cmd" ^
--ts_out=%JSOUTPATH% &

(echo /* tslint:disable */) >%JSOUTPATH%\backendmessage_pb.js.new
type %JSOUTPATH%\backendmessage_pb.js >>%JSOUTPATH%\backendmessage_pb.js.new
move /y %JSOUTPATH%\backendmessage_pb.js.new %JSOUTPATH%\backendmessage_pb.js

%GOPATH%\bin\protoc ^
-I=. -I=%GOPATH%\src -I=%GOPATH%\src\github.com\gogo\protobuf\protobuf ^
--gogoslick_out=%GOPROJECTPATH%\clientmessage\ ^
--proto_path=%GOPROJECTPATH%\clientmessage\ clientmessage.proto &

%GOPATH%\bin\protoc ^
--js_out=import_style=commonjs,binary:%JSOUTPATH% ^
--proto_path=%GOPROJECTPATH%\clientmessage\ clientmessage.proto ^
--plugin="protoc-gen-ts=%APPDATA%\npm\protoc-gen-ts.cmd" ^
--ts_out=%JSOUTPATH% &

(echo /* tslint:disable */) >%JSOUTPATH%\clientmessage_pb.js.new
type %JSOUTPATH%\clientmessage_pb.js >>%JSOUTPATH%\clientmessage_pb.js.new
move /y %JSOUTPATH%\clientmessage_pb.js.new %JSOUTPATH%\clientmessage_pb.js

%GOPATH%\bin\protoc ^
-I=. -I=%GOPATH%\src -I=%GOPATH%\src\github.com\gogo\protobuf\protobuf ^
--gogoslick_out=%GOPROJECTPATH%\frontendmessage\ ^
--proto_path=%GOPROJECTPATH%\frontendmessage\ frontendmessage.proto &

%GOPATH%\bin\protoc ^
--js_out=import_style=commonjs,binary:%JSOUTPATH% ^
--proto_path=%GOPROJECTPATH%\frontendmessage\ frontendmessage.proto ^
--plugin="protoc-gen-ts=%APPDATA%\npm\protoc-gen-ts.cmd" ^
--ts_out=%JSOUTPATH% &

(echo /* tslint:disable */) >%JSOUTPATH%\frontendmessage_pb.js.new
type %JSOUTPATH%\frontendmessage_pb.js >>%JSOUTPATH%\frontendmessage_pb.js.new
move /y %JSOUTPATH%\frontendmessage_pb.js.new %JSOUTPATH%\frontendmessage_pb.js