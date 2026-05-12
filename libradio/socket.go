package libradio

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"time"

	pb "github.com/pointnoreturn/snake/github.com/meshtastic/go/generated"
	"google.golang.org/protobuf/proto"
)

// TODO: meshtastic stuff here on Socket primitive?
const start1 = byte(0x94)
const start2 = byte(0xc3)
const headerLen = 4
const maxToFromRadioSzie = 512
const broadcastAddr = "^all"
const localAddr = "^local"
const defaultHopLimit = 3
const broadcastNum = 0xffffffff

// Socket holds the port and serial io.ReadWriteCloser struct to maintain one serial connection
type Socket struct {
	streamer streamer
	nodeNum  uint32
}

// Init initializes the Serial connection for the radio
func (r *Socket) Init(port string) error {

	streamer := streamer{}
	err := streamer.Init(port)
	if err != nil {
		return err
	}
	r.streamer = streamer

	return nil
}

// SendPacket takes a protbuf packet, construct the appropriate header and sends it to the radio
func (r *Socket) SendPacket(protobufPacket []byte) (err error) {

	packageLength := len(string(protobufPacket))

	header := []byte{start1, start2, byte(packageLength>>8) & 0xff, byte(packageLength) & 0xff}

	radioPacket := append(header, protobufPacket...)
	err = r.streamer.Write(radioPacket)
	if err != nil {
		return err
	}

	return

}

func (r *Socket) SendPacketContext(
	ctx context.Context,
	protobufPacket []byte,
) error {

	packageLength := len(protobufPacket)

	header := []byte{
		start1,
		start2,
		byte(packageLength>>8) & 0xff,
		byte(packageLength) & 0xff,
	}

	radioPacket := append(header, protobufPacket...)

	return r.streamer.WriteContext(ctx, radioPacket)
}

// ReadResponse reads any responses in the serial port, convert them to a FromRadio protobuf and return
func (r *Socket) ReadResponse(timeout bool) (FromRadioPackets []*pb.FromRadio, err error) {

	b := make([]byte, 1)

	emptyByte := make([]byte, 0)
	processedBytes := make([]byte, 0)
	repeatByteCounter := 0
	previousByte := make([]byte, 1)
	/************************************************************************************************
	* Process the returned data byte by byte until we have a valid command
	* Each command will come back with [START1, START2, PROTOBUF_PACKET]
	* where the protobuf packet is sent in binary. After reading START1 and START2
	* we use the next bytes to find the length of the packet.
	* After finding the length the looop continues to gather bytes until the length of the gathered
	* bytes is equal to the packet length plus the header
	 */
	for {
		err := r.streamer.Read(b)
		// fmt.Printf("Byte: %q\n", b)
		if bytes.Equal(b, previousByte) {
			repeatByteCounter++
		} else {
			repeatByteCounter = 0
		}
		// Only break on repeated bytes if we're not in the middle of reading a valid packet
		shouldBreakOnRepeat := repeatByteCounter > 20 && (len(processedBytes) < headerLen)

		if err == io.EOF || shouldBreakOnRepeat || errors.Is(err, os.ErrDeadlineExceeded) {
			break
		} else if err != nil {
			return nil, err
		}
		copy(previousByte, b)

		if len(b) > 0 {

			pointer := len(processedBytes)

			processedBytes = append(processedBytes, b...)

			if pointer == 0 {
				if b[0] != start1 {
					processedBytes = emptyByte
				}
			} else if pointer == 1 {
				if b[0] != start2 {
					processedBytes = emptyByte
				}
			} else if pointer >= headerLen {
				packetLength := int(processedBytes[2])<<8 | int(processedBytes[3])

				if pointer == headerLen {
					if packetLength > maxToFromRadioSzie {
						processedBytes = emptyByte
					}
				}

				if len(processedBytes) != 0 && pointer+1 == packetLength+headerLen {
					fromRadio := pb.FromRadio{}
					if err := proto.Unmarshal(processedBytes[headerLen:], &fromRadio); err != nil {
						return nil, err
					}
					FromRadioPackets = append(FromRadioPackets, &fromRadio)
					processedBytes = emptyByte
				}
			}

		} else {
			break
		}

	}

	return FromRadioPackets, nil

}

// ReadResponse reads any responses in the serial port, convert them to a FromRadio protobuf and return
func (r *Socket) ReadResponseContext(ctx context.Context, timeout bool) (FromRadioPackets []*pb.FromRadio, err error) {
	readCtx, cancel := context.WithTimeout(
		ctx,
		5*time.Second,
	)
	defer cancel()

	b := make([]byte, 1)

	emptyByte := make([]byte, 0)
	processedBytes := make([]byte, 0)
	repeatByteCounter := 0
	previousByte := make([]byte, 1)
	/************************************************************************************************
	* Process the returned data byte by byte until we have a valid command
	* Each command will come back with [START1, START2, PROTOBUF_PACKET]
	* where the protobuf packet is sent in binary. After reading START1 and START2
	* we use the next bytes to find the length of the packet.
	* After finding the length the looop continues to gather bytes until the length of the gathered
	* bytes is equal to the packet length plus the header
	 */
	for {

		err := r.streamer.ReadContext(readCtx, b)
		// fmt.Printf("Byte: %q\n", b)
		if bytes.Equal(b, previousByte) {
			repeatByteCounter++
		} else {
			repeatByteCounter = 0
		}
		// Only break on repeated bytes if we're not in the middle of reading a valid packet
		shouldBreakOnRepeat := repeatByteCounter > 20 && (len(processedBytes) < headerLen)

		if errors.Is(err, context.DeadlineExceeded) {
			err = nil
			if len(processedBytes) > 0 { // in the middle of reading packet
				// Hmm we would be able to recover in this case and continue using socket.
			}
			return FromRadioPackets, nil
		} else if err == io.EOF || shouldBreakOnRepeat || errors.Is(err, context.Canceled) {
			break
		} else if err != nil {
			fmt.Println("return err 1")
			return nil, err
		}
		copy(previousByte, b)

		if len(b) > 0 {

			pointer := len(processedBytes)

			processedBytes = append(processedBytes, b...)

			if pointer == 0 {
				if b[0] != start1 {
					processedBytes = emptyByte
				}
			} else if pointer == 1 {
				if b[0] != start2 {
					processedBytes = emptyByte
				}
			} else if pointer >= headerLen {
				packetLength := int(processedBytes[2])<<8 | int(processedBytes[3])

				if pointer == headerLen {
					if packetLength > maxToFromRadioSzie {
						processedBytes = emptyByte
					}
				}

				if len(processedBytes) != 0 && pointer+1 == packetLength+headerLen {
					fromRadio := pb.FromRadio{}
					if err := proto.Unmarshal(processedBytes[headerLen:], &fromRadio); err != nil {
						fmt.Println("return err 2")
						return nil, err
					}
					FromRadioPackets = append(FromRadioPackets, &fromRadio)
					processedBytes = emptyByte
				}
			}

		} else {
			break
		}

	}

	return FromRadioPackets, nil

}

// createAdminPacket builds a admin message packet to send to the radio
func (r *Socket) createAdminPacket(nodeNum uint32, payload []byte) (packetOut []byte, err error) {

	radioMessage := pb.ToRadio{
		PayloadVariant: &pb.ToRadio_Packet{
			Packet: &pb.MeshPacket{
				To:      nodeNum,
				WantAck: true,
				PayloadVariant: &pb.MeshPacket_Decoded{
					Decoded: &pb.Data{
						Payload:      payload,
						Portnum:      pb.PortNum_ADMIN_APP,
						WantResponse: true,
					},
				},
			},
		},
	}

	packetOut, err = proto.Marshal(&radioMessage)
	if err != nil {
		return nil, err
	}

	return

}

func (r *Socket) PostInit(myNodeNum uint32) {
	r.nodeNum = myNodeNum
}

// firmware consts for want_config_id:
// #define SPECIAL_NONCE_ONLY_CONFIG 69420
// #define SPECIAL_NONCE_ONLY_NODES 69421 // ( ͡° ͜ʖ ͡°)
const ConfigId_OnlyNodes = 69421
const ConfigId_ConfigOnly = 69420

// GetRadioInfo retrieves information from the radio including config and adjacent Node information
// Right after TCP dial is finished
func (r *Socket) SendWantConfigId(ctx context.Context, id uint32) (radioResponses []*pb.FromRadio, err error) {
	// Send first request for Radio and Node information
	nodeInfo := pb.ToRadio{PayloadVariant: &pb.ToRadio_WantConfigId{WantConfigId: id}} // only want self node info

	out, err := proto.Marshal(&nodeInfo)
	if err != nil {
		return nil, err
	}

	err = r.SendPacketContext(ctx, out)
	if err != nil {
		return nil, err
	}

	fmt.Println("SendPacketContext success")

	radioResponses, err = r.ReadResponseContext(ctx, true)
	fmt.Printf("ReadResponseContext returned with err=%v\n", err)
	if err != nil {
		return nil, err
	}

	if len(radioResponses) == 0 {
		return nil, errors.New("failed to get radio info")
	}
	return // but radioResponses contain 200+ nodes from nodedb

}
func (r *Socket) SendHeartbeat(nonce uint32) (err error) {
	// Send first request for Radio and Node information
	nodeInfo := pb.ToRadio{PayloadVariant: &pb.ToRadio_Heartbeat{
		Heartbeat: &pb.Heartbeat{
			Nonce: nonce,
		},
	}} // only want self node info

	out, err := proto.Marshal(&nodeInfo)
	if err != nil {
		return err
	}

	return r.SendPacket(out)
}

// GetRadioInfo retrieves information from the radio including config and adjacent Node information
func (r *Socket) GetRadioInfo() (radioResponses []*pb.FromRadio, err error) {
	// Send first request for Radio and Node information
	nodeInfo := pb.ToRadio{PayloadVariant: &pb.ToRadio_WantConfigId{WantConfigId: 42}}

	out, err := proto.Marshal(&nodeInfo)
	if err != nil {
		return nil, err
	}

	r.SendPacket(out)

	checks := 0

	radioResponses, err = r.ReadResponse(true)

	for checks < 5 && len(radioResponses) == 0 {
		radioResponses, err = r.ReadResponse(true)
		if err != nil {
			return nil, err
		}

		checks++
		time.Sleep(1 * time.Second)
	}

	if len(radioResponses) == 0 {
		return nil, errors.New("failed to get radio info")
	}
	return

}

// SendTextMessage sends a free form text message to other radios
func (r *Socket) SendTextMessage(message string, to int64, channel int64) error {
	var address int64
	if to == 0 {
		address = broadcastNum
	} else {
		address = to
	}

	// This constant is defined in Constants_DATA_PAYLOAD_LEN, but not in a friendly way to use
	if len(message) > 240 {
		return errors.New("message too large")
	}

	rand.Seed(time.Now().UnixNano())
	packetID := rand.Intn(2386828-1) + 1

	radioMessage := pb.ToRadio{
		PayloadVariant: &pb.ToRadio_Packet{
			Packet: &pb.MeshPacket{
				To:      uint32(address),
				WantAck: true,
				Id:      uint32(packetID),
				Channel: uint32(channel),
				PayloadVariant: &pb.MeshPacket_Decoded{
					Decoded: &pb.Data{
						Payload: []byte(message),
						Portnum: pb.PortNum_TEXT_MESSAGE_APP,
					},
				},
			},
		},
	}

	out, err := proto.Marshal(&radioMessage)
	if err != nil {
		return err
	}

	if err := r.SendPacket(out); err != nil {
		return err
	}

	return nil

}

// SetRadioOwner sets the owner of the radio visible on the public mesh
func (r *Socket) SetRadioOwner(name string) error {

	if len(name) <= 2 {
		return errors.New("name too short")
	}

	adminPacket := pb.AdminMessage{
		PayloadVariant: &pb.AdminMessage_SetOwner{
			SetOwner: &pb.User{
				LongName:  name,
				ShortName: name[:3],
			},
		},
	}

	out, err := proto.Marshal(&adminPacket)
	if err != nil {
		return err
	}

	nodeNum := r.nodeNum

	packet, err := r.createAdminPacket(nodeNum, out)
	if err != nil {
		return err
	}

	if err := r.SendPacket(packet); err != nil {
		return err
	}

	return nil
}

// SetModemMode sets the channel modem setting to be fast or slow
func (r *Socket) SetModemMode(mode string) error {

	var modemSetting pb.Config_LoRaConfig_ModemPreset

	if mode == "lf" {
		modemSetting = pb.Config_LoRaConfig_LONG_FAST
	} else if mode == "ls" {
		modemSetting = pb.Config_LoRaConfig_LONG_SLOW
	} else if mode == "vls" {
		modemSetting = pb.Config_LoRaConfig_VERY_LONG_SLOW
	} else if mode == "ms" {
		modemSetting = pb.Config_LoRaConfig_MEDIUM_SLOW
	} else if mode == "mf" {
		modemSetting = pb.Config_LoRaConfig_MEDIUM_FAST
	} else if mode == "sl" {
		modemSetting = pb.Config_LoRaConfig_SHORT_SLOW
	} else if mode == "sf" {
		modemSetting = pb.Config_LoRaConfig_SHORT_FAST
	} else if mode == "lm" {
		modemSetting = pb.Config_LoRaConfig_LONG_MODERATE
	}

	adminPacket := pb.AdminMessage{
		PayloadVariant: &pb.AdminMessage_SetConfig{
			SetConfig: &pb.Config{
				PayloadVariant: &pb.Config_Lora{
					Lora: &pb.Config_LoRaConfig{
						ModemPreset: modemSetting,
					},
				},
			},
		},
	}

	out, err := proto.Marshal(&adminPacket)
	if err != nil {
		return err
	}

	nodeNum := r.nodeNum

	packet, err := r.createAdminPacket(nodeNum, out)
	if err != nil {
		return err
	}

	if err := r.SendPacket(packet); err != nil {
		return err
	}

	return nil

}

// SetLocation sets a fixed location for the radio
func (r *Socket) SetLocation(lat int32, long int32, alt int32) error {

	positionPacket := pb.Position{
		LatitudeI:  proto.Int32(lat),
		LongitudeI: proto.Int32(long),
		Altitude:   proto.Int32(alt),
	}

	out, err := proto.Marshal(&positionPacket)
	if err != nil {
		return err
	}

	nodeNum := r.nodeNum

	radioMessage := pb.ToRadio{
		PayloadVariant: &pb.ToRadio_Packet{
			Packet: &pb.MeshPacket{
				To:      nodeNum,
				WantAck: true,
				PayloadVariant: &pb.MeshPacket_Decoded{
					Decoded: &pb.Data{
						Payload:      out,
						Portnum:      pb.PortNum_POSITION_APP,
						WantResponse: true,
					},
				},
			},
		},
	}

	packet, err := proto.Marshal(&radioMessage)
	if err != nil {
		return err
	}

	if err := r.SendPacket(packet); err != nil {
		return err
	}

	return nil
}

// Send a factory reset command to the radio
func (r *Socket) FactoryRest() error {
	adminPacket := pb.AdminMessage{
		PayloadVariant: &pb.AdminMessage_FactoryResetDevice{
			FactoryResetDevice: 1, // FIXME: check if this must be some value..?
		},
	}
	out, err := proto.Marshal(&adminPacket)
	if err != nil {
		return err
	}

	nodeNum := r.nodeNum

	packet, err := r.createAdminPacket(nodeNum, out)
	if err != nil {
		return err
	}

	if err := r.SendPacket(packet); err != nil {
		return err
	}

	return nil
}

// Close closes the serial port. Added so users can defer the close after opening
func (r *Socket) Close() {
	r.streamer.Close()
}
