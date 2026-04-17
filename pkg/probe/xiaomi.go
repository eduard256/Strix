package probe

import (
	"context"
	"encoding/binary"
	"net"
	"time"
)

// miIO hello packet -- 32 bytes. Stock Xiaomi/Mijia devices listen on
// UDP:54321 and reply with the same magic 0x2131 + their device_id + stamp.
// Newer firmwares always return 0xFF in the token field, regardless of
// pairing status -- real token is only available via Mi Cloud API.
var xiaomiHello = []byte{
	0x21, 0x31, 0x00, 0x20,
	0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff,
}

// ProbeXiaomi sends miIO hello to ip:54321 and checks the reply magic.
// Returns nil, nil if the device is not a Xiaomi miIO device.
func ProbeXiaomi(ctx context.Context, ip string) (*XiaomiResult, error) {
	conn, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(100 * time.Millisecond)
	}
	_ = conn.SetDeadline(deadline)

	addr := &net.UDPAddr{IP: net.ParseIP(ip), Port: 54321}
	if _, err = conn.WriteTo(xiaomiHello, addr); err != nil {
		return nil, err
	}

	buf := make([]byte, 64)
	n, _, err := conn.ReadFrom(buf)
	if err != nil || n < 32 {
		return nil, nil
	}

	// magic must be 0x2131 -- unique miIO header
	if buf[0] != 0x21 || buf[1] != 0x31 {
		return nil, nil
	}

	return &XiaomiResult{
		DeviceID: binary.BigEndian.Uint32(buf[8:12]),
		Stamp:    binary.BigEndian.Uint32(buf[12:16]),
	}, nil
}
