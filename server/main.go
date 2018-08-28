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
	"net"
	"os"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/golang/protobuf/proto"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"github.com/grpc-ecosystem/go-grpc-middleware/tags"
	log "github.com/sirupsen/logrus"

	pb "github.com/phriscage/beer-likes/beerlikes"
)

var (
	tls        = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	certFile   = flag.String("cert_file", "", "The TLS cert file")
	keyFile    = flag.String("key_file", "", "The TLS key file")
	jsonDBFile = flag.String("json_db_file", "testdata/beer_likes_db.json", "A json file containing a list of features")
	port       = flag.Int("port", 10000, "The server port")
	host       = flag.String("host", "127.0.0.1", "The server host ip")
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
	if query == nil {
		return &pb.Like{}, status.Error(codes.InvalidArgument, fmt.Sprintf("'%+v' is not valid", query))
	}
	for _, item := range s.savedLikes {
		if item.Id == query.Id {
			return item, nil
		}
	}
	// No like was found, return an unnamed like
	return &pb.Like{}, status.Error(codes.NotFound, fmt.Sprintf("'%+v' was not found", query))
}

// ListLikes lists all likes contained within the given bounding Like.
func (s *beerLikesServer) ListLikes(query *pb.LikesQuery, stream pb.BeerLikes_ListLikesServer) error {
	if query.RefType == nil {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("'%+v' is not valid", query))
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
		log.Warnf("Failed to load default likes: %v", err)
	}
	if err := json.Unmarshal(file, &s.savedLikes); err != nil {
		log.Warnf("Failed to load default likes: %v", err)
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

func defaultServerOpts() []grpc.ServerOption {
	return []grpc.ServerOption{}
}

// withDuration returns the duration of a grpc connection in nanoseconds
func withDuration(duration time.Duration) (key string, value interface{}) {
	return "grpc.time_ns", duration.Nanoseconds()
}

func main() {
	flag.Parse()
	host_port := fmt.Sprintf("%s:%d", *host, *port)
	lis, err := net.Listen("tcp", host_port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	var opts []grpc.ServerOption

	logrusEntry := log.NewEntry(log.StandardLogger())
	logOpts := []grpc_logrus.Option{
		grpc_logrus.WithDurationField(withDuration),
	}

	opts = append(
		opts,
		grpc_middleware.WithUnaryServerChain(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_logrus.UnaryServerInterceptor(logrusEntry, logOpts...)),
		grpc_middleware.WithStreamServerChain(
			grpc_ctxtags.StreamServerInterceptor(),
			grpc_logrus.StreamServerInterceptor(logrusEntry, logOpts...)),
	)

	log.Infof("Starting grpc server on '%s'", host_port)
	grpcServer := grpc.NewServer(append(defaultServerOpts(), opts...)...)

	pb.RegisterBeerLikesServer(grpcServer, newServer())
	grpcServer.Serve(lis)
	log.Infof("Stopping grpc server...")
}
