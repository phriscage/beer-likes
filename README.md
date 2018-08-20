# beer-likes

## Build

Execute this command 

        protoc -I beerlikes --go_out=plugins=grpc:beerlikes beerlikes/beer_likes.proto

## Run Server and Client

In one terminal: 

         go run server/server.go


In the other terminal:

        go run client/client.go
