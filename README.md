# beer-likes

## Development 
### Skaffold

Build and deploy with [Skaffold](https://github.com/GoogleContainerTools/skaffold)

* Install Skaffold [here](https://github.com/GoogleContainerTools/skaffold#installation)

* Change the *imageName* and *projectId* in the [skaffold.yaml](skaffold.yaml)

In one terminal: 

        skaffold dev

In the other terminal:

        go run client/client.go -server_addr $EXTERNAL_IP:10000


### gRPC Protocol Buffer definition

Execute this command 

        protoc -I beerlikes --go_out=plugins=grpc:beerlikes beerlikes/beer_likes.proto


### Local Server and Client

In one terminal: 

        go run server/server.go


In the other terminal:

        go run client/client.go
