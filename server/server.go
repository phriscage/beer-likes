/*
 *
 * Copyright 2018 Chris Page <phriscage@gmail.com>
 *
 */

//go:generate protoc -I ../beerlikes --go_out=plugins=grpc:../beerlikes ../beerlikes/beer_likes.proto

// Package main implements a simple gRPC server that demonstrates how to use gRPC-Go libraries
// to perform unary and server streaming
//
// It implements the Beer Likes service whose definition can be found in beerlikes/beer_likes.proto.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/testdata"

	"github.com/golang/protobuf/proto"

	// pb "github.com/phriscage/beer-likes/beerlikes"
	pb "../beerlikes"
)

var (
	tls        = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	certFile   = flag.String("cert_file", "", "The TLS cert file")
	keyFile    = flag.String("key_file", "", "The TLS key file")
	jsonDBFile = flag.String("json_db_file", "testdata/beer_likes_db.json", "A json file containing a list of features")
	port       = flag.Int("port", 10000, "The server port")
)

type beerLikesServer struct {
	savedLikes []*pb.Like // read-only after initialized
	//mu         sync.Mutex // protects routeNotes
	//routeNotes map[string][]*pb.RouteNote
}

// GetLike returns the feature at the given Like.
func (s *beerLikesServer) GetLike(ctx context.Context, query *pb.LikeQuery) (*pb.Like, error) {
	if query == nil {
		return &pb.Like{}, nil
	}
	for _, item := range s.savedLikes {
		if item.Id == query.Id {
			return item, nil
		}
	}
	// No like was found, return an unnamed like
	return &pb.Like{}, nil
}

// ListLikes lists all likes contained within the given bounding Like.
func (s *beerLikesServer) ListLikes(query *pb.LikesQuery, stream pb.BeerLikes_ListLikesServer) error {
	if query.RefType == nil {
		return nil
	}
	for _, item := range s.savedLikes {
		if proto.Equal(item.RefType, query.RefType) {
			if err := stream.Send(item); err != nil {
				return err
			}
		}
	}
	return nil
}

// GetLikesSummary batch fetches the likes contained within the given bounding Like.
func (s *beerLikesServer) GetLikesSummary(ctx context.Context, query *pb.LikesQuery) (*pb.LikesSummary, error) {
	var total int32
	var likes []*pb.Like
	startTime := time.Now()
	for _, item := range s.savedLikes {
		if proto.Equal(item.RefType, query.RefType) {
			if item.Liked {
				total++
			} else {
				total--
			}
			likes = append(likes, item)
		}
	}
	endTime := time.Now()
	return &pb.LikesSummary{
		Likes:       likes,
		Total:       total,
		ElapsedTime: uint64(endTime.Sub(startTime)),
	}, nil
}

// loadLikes loads likes from a JSON file.
func (s *beerLikesServer) loadLikes(filePath string) {
	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Failed to load default likes: %v", err)
	}
	if err := json.Unmarshal(file, &s.savedLikes); err != nil {
		log.Fatalf("Failed to load default likes: %v", err)
	}
}

func serialize(Like *pb.Like) string {
	return fmt.Sprintf("%d %d", Like.RefType, Like.Id)
}

func newServer() *beerLikesServer {
	s := &beerLikesServer{}
	s.loadLikes(*jsonDBFile)
	return s
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	if *tls {
		if *certFile == "" {
			*certFile = testdata.Path("server1.pem")
		}
		if *keyFile == "" {
			*keyFile = testdata.Path("server1.key")
		}
		creds, err := credentials.NewServerTLSFromFile(*certFile, *keyFile)
		if err != nil {
			log.Fatalf("Failed to generate credentials %v", err)
		}
		opts = []grpc.ServerOption{grpc.Creds(creds)}
	}
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterBeerLikesServer(grpcServer, newServer())
	grpcServer.Serve(lis)
}
