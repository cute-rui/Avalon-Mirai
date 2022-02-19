package main

import (
    "avalon-mirai/src/mirai"
    "google.golang.org/grpc"
    "google.golang.org/grpc/keepalive"
    "log"
    "net"
    "sync"
    "time"
)

var kaep = keepalive.EnforcementPolicy{
    MinTime:             5 * time.Second, // If a client pings more than once every 5 seconds, terminate the connection
    PermitWithoutStream: true,            // Allow pings even when there are no active streams
}

var kasp = keepalive.ServerParameters{
    Time:    5 * time.Second, // Ping the client if it is idle for 5 seconds to ensure the connection is still active
    Timeout: 1 * time.Second, // Wait 1 second for the ping ack before assuming the connection is dead
}

func main() {
    lis, err := net.Listen(`tcp`, Conf.GetString(`service.addr`))
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }
    
    grpcServer := grpc.NewServer()
    mirai.RegisterMiraiAgentServer(grpcServer, &MiraiAgentServiceServer{WebsocketServers: &sync.Map{}})
    grpcServer.Serve(lis)
}
