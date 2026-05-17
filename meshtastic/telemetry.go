package meshtastic

import (
	"context"
	"time"

	pb "github.com/pointnoreturn/monitor/github.com/meshtastic/go/generated"
	"google.golang.org/protobuf/proto"
)

func RequestTelemetry(ctx context.Context, stream Writer, to uint32, _type *pb.Telemetry) (requestId uint32, err error) {
	requestId = uint32(time.Now().UnixNano() & 0x3FF)

	data, err := proto.Marshal(_type)
	if err != nil {
		return 0, err
	}

	toRadio := pb.ToRadio{
		PayloadVariant: &pb.ToRadio_Packet{
			Packet: &pb.MeshPacket{
				Id:       requestId,
				Channel:  0,
				HopLimit: 0,
				To:       to,
				PayloadVariant: &pb.MeshPacket_Decoded{
					Decoded: &pb.Data{
						RequestId:    requestId,
						Payload:      data,
						WantResponse: true,
						Portnum:      pb.PortNum_TELEMETRY_APP,
					},
				},
			},
		},
	}

	return requestId, stream.WritePacket(ctx, &toRadio)
}
