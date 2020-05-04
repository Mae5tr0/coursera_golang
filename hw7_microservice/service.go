package main

import (
	"context"
	"encoding/json"
	"net"
	"strings"
	"sync"
	"time"

	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type bizServer struct {
}

type adminServer struct {
	mu       sync.Mutex
	eventBus []chan *Event
}

func (s *bizServer) Check(ctx context.Context, nothing *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

func (s *bizServer) Add(ctx context.Context, nothing *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

func (s *bizServer) Test(ctx context.Context, nothing *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

func (s *adminServer) Logging(nothing *Nothing, server Admin_LoggingServer) error {
	in := s.Subscribe()

	for ev := range in {
		if err := server.Send(ev); err != nil {
			fmt.Printf("Sendint error: %v", err)
			return err
		}
	}

	return nil
}

func makeStat() Stat {
	return Stat{ByMethod: make(map[string]uint64), ByConsumer: make(map[string]uint64)}
}
func updateStat(stat *Stat, ev *Event) {
	if _, set := stat.ByMethod[ev.Method]; !set {
		stat.ByMethod[ev.Method] = 0
	}
	stat.ByMethod[ev.Method]++
	if _, set := stat.ByConsumer[ev.Consumer]; !set {
		stat.ByConsumer[ev.Consumer] = 0
	}
	stat.ByConsumer[ev.Consumer]++
	stat.Timestamp = time.Now().Unix()
}

func (s *adminServer) Statistics(statInterval *StatInterval, server Admin_StatisticsServer) error {
	ticker := time.NewTicker(time.Duration(statInterval.IntervalSeconds) * time.Second)
	defer ticker.Stop()

	in := s.Subscribe()
	stat := makeStat()

	for {
		select {
		case ev := <-in:
			updateStat(&stat, ev)
		case <-ticker.C:
			if err := server.Send(&stat); err != nil {
				fmt.Printf("Sendint error: %v", err)
				return nil
			}
			stat = makeStat()
		}
	}
	return nil
}

func (s *adminServer) Subscribe() chan *Event {
	ch := make(chan *Event)
	s.mu.Lock()
	s.eventBus = append(s.eventBus, ch)
	s.mu.Unlock()

	return ch
}

func (s *adminServer) Broadcast(event *Event) {
	s.mu.Lock()
	for _, ch := range s.eventBus {
		ch <- event
	}
	s.mu.Unlock()
}

// https://github.com/grpc-ecosystem/go-grpc-middleware/blob/master/chain.go
func ChainUnaryServer(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	n := len(interceptors)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		chainer := func(currentInter grpc.UnaryServerInterceptor, currentHandler grpc.UnaryHandler) grpc.UnaryHandler {
			return func(currentCtx context.Context, currentReq interface{}) (interface{}, error) {
				return currentInter(currentCtx, currentReq, info, currentHandler)
			}
		}

		chainedHandler := handler
		for i := n - 1; i >= 0; i-- {
			chainedHandler = chainer(interceptors[i], chainedHandler)
		}

		return chainedHandler(ctx, req)
	}
}
func ChainStreamServer(interceptors ...grpc.StreamServerInterceptor) grpc.StreamServerInterceptor {
	n := len(interceptors)

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		chainer := func(currentInter grpc.StreamServerInterceptor, currentHandler grpc.StreamHandler) grpc.StreamHandler {
			return func(currentSrv interface{}, currentStream grpc.ServerStream) error {
				return currentInter(currentSrv, currentStream, info, currentHandler)
			}
		}

		chainedHandler := handler
		for i := n - 1; i >= 0; i-- {
			chainedHandler = chainer(interceptors[i], chainedHandler)
		}

		return chainedHandler(srv, ss)
	}
}

// Authorization
func UnaryAuthInterceptor(acl map[string][]string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (interface{}, error) {
		err := authorize(ctx, info.FullMethod, acl)
		if err != nil {
			return nil, err
		}

		return handler(ctx, req)
	}
}
func StreamAuthInterceptor(acl map[string][]string) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()
		err := authorize(ctx, info.FullMethod, acl)
		if err != nil {
			return err
		}

		return handler(srv, ss)
	}
}
func authorize(ctx context.Context, fullMethod string, acl map[string][]string) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "Invalid metadata")
	}
	consumerList, ok := md["consumer"]
	if !ok || len(consumerList) == 0 {
		return status.Error(codes.Unauthenticated, "Empty consumer")
	}
	consumer := consumerList[0]

	access := false
	permissions, ok := acl[consumer]
	if !ok {
		return status.Error(codes.Unauthenticated, "Unknown consumer")
	}

	reqInfo := strings.Split(fullMethod, "/")
	reqServer := reqInfo[1]
	reqMethod := reqInfo[2]
	for _, permission := range permissions {
		consumerPermission := strings.Split(permission, "/")
		if consumerPermission[1] != reqServer {
			continue
		}
		if consumerPermission[2] == "*" || consumerPermission[2] == reqMethod {
			return nil
		}
	}

	if !access {
		return status.Error(codes.Unauthenticated, "Unauthorized")
	}

	return nil
}

func getConsumer(ctx context.Context) string {
	md, _ := metadata.FromIncomingContext(ctx)
	consumerList, _ := md["consumer"]
	return consumerList[0]
}

func publishEvent(ctx context.Context, fullMethod string, server *adminServer) {
	server.Broadcast(&Event{Timestamp: time.Now().Unix(), Consumer: getConsumer(ctx), Method: fullMethod, Host: "127.0.0.1:8083"})
}

// Logging
func UnaryLogInterceptor(server *adminServer) grpc.UnaryServerInterceptor {
	return func(ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (interface{}, error) {
		publishEvent(ctx, info.FullMethod, server)

		return handler(ctx, req)
	}
}
func StreamLogInterceptor(server *adminServer) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		publishEvent(ss.Context(), info.FullMethod, server)

		return handler(srv, ss)
	}
}

func StartMyMicroservice(ctx context.Context, listenAddr string, aclJson string) error {
	acl := make(map[string][]string)
	err := json.Unmarshal([]byte(aclJson), &acl)
	if err != nil {
		return err
	}

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}

	adminServer := &adminServer{}
	bizServer := &bizServer{}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(
			ChainUnaryServer(
				UnaryAuthInterceptor(acl),
				UnaryLogInterceptor(adminServer),
			),
		),
		grpc.StreamInterceptor(
			ChainStreamServer(
				StreamAuthInterceptor(acl),
				StreamLogInterceptor(adminServer),
			),
		),
	)
	RegisterAdminServer(grpcServer, adminServer)
	RegisterBizServer(grpcServer, bizServer)

	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
	}()

	go func() {
		grpcServer.Serve(lis)
	}()

	return nil
}
