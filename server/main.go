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
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
	"os"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/testdata"

	"github.com/golang/protobuf/proto"

	pb "github.com/phriscage/beer-likes/beerlikes"
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

// Init
func init() {
	// Log as JSON instead of the default ASCII formatter.
	//log.SetFormatter(&log.JSONFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the debug severity or above.
	log.SetLevel(log.DebugLevel)
}

// GetLike returns the feature at the given Like.
func (s *beerLikesServer) GetLike(ctx context.Context, query *pb.LikeQuery) (*pb.Like, error) {
	log.Debugf("GetLike query: '%v'", query)
	if query == nil {
		return &pb.Like{}, status.Error(codes.InvalidArgument, fmt.Sprintf("%+v is not valid", query))
	}
	for _, item := range s.savedLikes {
		if item.Id == query.Id {
			return item, nil
		}
	}
	// No like was found, return an unnamed like
	return &pb.Like{}, status.Error(codes.NotFound, fmt.Sprintf("%+v was not found", query))
}

// ListLikes lists all likes contained within the given bounding Like.
func (s *beerLikesServer) ListLikes(query *pb.LikesQuery, stream pb.BeerLikes_ListLikesServer) error {
	log.Debugf("ListLikes query: '%v'", query)
	if query.RefType == nil {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("%+v is not valid", query))
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
	log.Debugf("GetLikesSummary query: '%v'", query)
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

// Authorization unary interceptor function to handle authorize per RPC call
func serverInterceptor(ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()
	// Skip authorize when GetJWT is requested
	//if info.FullMethod != "/proto.EventStoreService/GetJWT" {
	//if err := authorize(ctx); err != nil {
	//return nil, err
	//}
	//}

	// Calls the handler
	h, err := handler(ctx, req)

	// Extract the metadata headers and peer info
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		err = status.Errorf(codes.InvalidArgument, "Retrieving metadata has failed")
		log.Fatal(err)
		return nil, err
	}
	pr, ok := peer.FromContext(ctx)
	if !ok {
		err = status.Errorf(codes.InvalidArgument, "Retrieving peer has failed")
		log.Fatal(err)
		return nil, err
	}
	if pr.Addr == net.Addr(nil) {
		err = status.Errorf(codes.InvalidArgument, "Failed to get peer address")
		log.Fatal(err)
		return nil, err
	}

	// request/response logging
	fields := log.Fields{
		"remote_ip":    pr.Addr,
		"current_time": time.Now().UTC().Format(time.RFC3339),
		"full_method":  info.FullMethod,
		"user-agent":   md["user-agent"][0],
		"status_code":  codes.OK,
		"elapsed_time": time.Since(start),
	}
	log.WithFields(fields).Info()

	return h, err
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", *port))
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

	opts = []grpc.ServerOption{grpc.UnaryInterceptor(serverInterceptor)}
	log.Infof("Starting grpc server on '%d'", port)
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterBeerLikesServer(grpcServer, newServer())
	grpcServer.Serve(lis)
	log.Infof("Stopping grpc server...")
}
