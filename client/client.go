/*
 *
 * Copyright 2018 Chris Page <phriscage@gmail.com>
 *
 */

// Package main implements a simple gRPC client that demonstrates how to use gRPC-Go libraries
// to perform unary and server streaming
//
// It interacts with the beer likes service whose definition can be found in beerlikes/beer_likes.proto.
package main

import (
	"flag"
	"log"
	"time"

	pb "../beerlikes"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/testdata"
)

var (
	tls                = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	caFile             = flag.String("ca_file", "", "The file containning the CA root cert file")
	serverAddr         = flag.String("server_addr", "127.0.0.1:10000", "The server address in the format of host:port")
	serverHostOverride = flag.String("server_host_override", "x.test.youtube.com", "The server name use to verify the hostname returned by TLS handshake")
)

// printLike gets the like for the given point.
func printLike(client pb.BeerLikesClient, query *pb.LikeQuery) {
	log.Printf("Getting like for like (%s)", query)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	like, err := client.GetLike(ctx, query)
	if err != nil {
		log.Fatalf("%v.GetLikes(_) = _, %v: ", client, err)
	}
	log.Println(like)
}
func main() {
	flag.Parse()
	var opts []grpc.DialOption
	if *tls {
		if *caFile == "" {
			*caFile = testdata.Path("ca.pem")
		}
		creds, err := credentials.NewClientTLSFromFile(*caFile, *serverHostOverride)
		if err != nil {
			log.Fatalf("Failed to create TLS credentials %v", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}
	conn, err := grpc.Dial(*serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()
	client := pb.NewBeerLikesClient(conn)

	// Looking for a valid like
	// printLike(client, &pb.LikeQuery{RefType: &pb.RefType{Name: "beer", Id: "1"}})
	printLike(client, &pb.LikeQuery{Id: "3e8f9d58-4148-4809-9392-63e90fbc8280"})

	// Like missing.
	printLike(client, &pb.LikeQuery{})

}
