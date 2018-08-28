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
	"fmt"
	"io"
	"log"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/testdata"

	pb "github.com/phriscage/beer-likes/beerlikes"
)

var (
	tls                = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	caFile             = flag.String("ca_file", "", "The file containning the CA root cert file")
	port               = flag.Int("port", 10000, "The server port")
	host               = flag.String("host", "127.0.0.1", "The server host ip")
	serverHostOverride = flag.String("server_host_override", "x.test.youtube.com", "The server name use to verify the hostname returned by TLS handshake")
)

// printLike gets the like for the given point.
func printLike(client pb.BeerLikesClient, query *pb.LikeQuery) {
	log.Printf("Getting like for like (%s)", query)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var header, trailer metadata.MD // variable to store header and trailer
	like, err := client.GetLike(ctx, query, grpc.Header(&header), grpc.Trailer(&trailer))
	if err != nil {
		log.Printf("%v.GetLikes(_) = _, %v: ", client, err)
		return
	}
	log.Printf("res headers: %v", header)
	log.Printf("res trailer: %v", trailer)
	log.Println(like)
}

// printLikes lists all the likes within the given bounding RefType.
func printLikes(client pb.BeerLikesClient, query *pb.LikesQuery) {
	startTime := time.Now()
	log.Printf("Looking for all likes within %v", query)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stream, err := client.ListLikes(ctx, query)
	likesSummary := pb.LikesSummary{}
	if err != nil {
		log.Fatalf("%v.ListLikes(_) = _, %v", client, err)
	}
	// retrieve header
	header, err := stream.Header()
	// retrieve trailer
	trailer := stream.Trailer()
	log.Printf("res headers: %v", header)
	for {
		item, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("%v.ListLikes(_) = _, %v", client, err)
		}
		// log.Println(item)
		likesSummary.Likes = append(likesSummary.Likes, item)
		if item.Liked {
			likesSummary.Total++
		} else {
			likesSummary.Total--
		}
	}
	log.Printf("res trailer: %v", trailer)
	endTime := time.Now()
	likesSummary.ElapsedTime = uint64(endTime.Sub(startTime))
	log.Println(likesSummary)
}

// printLikesSummary lists all the likes within the given bounding RefType.
func printLikesSummary(client pb.BeerLikesClient, query *pb.LikesQuery) {
	log.Printf("Looking for all likes within %v", query)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	likesSummary, err := client.GetLikesSummary(ctx, query)
	if err != nil {
		log.Fatalf("%v.GetLikesSummary(_) = _, %v: ", client, err)
	}
	log.Println(likesSummary)
}

// Main
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
	host_port := fmt.Sprintf("%s:%d", *host, *port)
	conn, err := grpc.Dial(host_port, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()
	client := pb.NewBeerLikesClient(conn)

	md := metadata.Pairs(
		"key", "string value",
		"key-bin", string([]byte{96, 102}), // this binary data will be encoded (base64) before sending
		// and will be decoded after being transferred.
	)
	log.Printf("metadata: %v", md)
	printLike(client, &pb.LikeQuery{Id: "3e8f9d58-4148-4809-9392-63e90fbc8280"})

	// Like NotFound
	printLike(client, &pb.LikeQuery{Id: "123-abc"})

	// Like missing.
	printLike(client, &pb.LikeQuery{})

	// return all the likes for a given reftype
	printLikes(client, &pb.LikesQuery{
		RefType: &pb.RefType{Name: "beer", Id: "1"},
	})

	// return all the likes for a given reftype
	printLikesSummary(client, &pb.LikesQuery{
		RefType: &pb.RefType{Name: "beer", Id: "1"},
	})

	// return all the likes for an incorrect reftype
	printLikes(client, &pb.LikesQuery{
		RefType: &pb.RefType{Name: "beer", Id: "xyz"},
	})

}
