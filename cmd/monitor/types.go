package main

import (
	"context"

	pb "github.com/pointnoreturn/monitor/github.com/meshtastic/go/generated"
)

type Worker interface {
	HandlePacket(*pb.FromRadio)
	Init(ctx context.Context)
	Run(context.Context)
}
