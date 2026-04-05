package probe

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
)

const (
	hapService = "_hap._tcp.local."

	txtCategory    = "ci"
	txtDeviceID    = "id"
	txtModel       = "md"
	txtStatusFlags = "sf"

	statusPaired     = "0"
	categoryCamera   = "17"
	categoryDoorbell = "18"
)

// QueryHAP sends unicast mDNS query to ip:5353 for HomeKit service.
// Returns nil if device is not a HomeKit camera/doorbell.
func QueryHAP(ctx context.Context, ip string) (*MDNSResult, error) {
	msg := &dns.Msg{
		Question: []dns.Question{
			{Name: hapService, Qtype: dns.TypePTR, Qclass: dns.ClassINET},
		},
	}

	query, err := msg.Pack()
	if err != nil {
		return nil, err
	}

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

	addr := &net.UDPAddr{IP: net.ParseIP(ip), Port: 5353}
	if _, err = conn.WriteTo(query, addr); err != nil {
		return nil, err
	}

	buf := make([]byte, 1500)
	n, _, err := conn.ReadFrom(buf)
	if err != nil {
		return nil, nil // timeout = not a HomeKit device
	}

	var resp dns.Msg
	if err = resp.Unpack(buf[:n]); err != nil {
		return nil, nil
	}

	return parseHAPResponse(&resp)
}

// internals

func parseHAPResponse(msg *dns.Msg) (*MDNSResult, error) {
	records := make([]dns.RR, 0, len(msg.Answer)+len(msg.Extra))
	records = append(records, msg.Answer...)
	records = append(records, msg.Extra...)

	var ptrName string
	for _, rr := range records {
		if ptr, ok := rr.(*dns.PTR); ok && ptr.Hdr.Name == hapService {
			ptrName = ptr.Ptr
			break
		}
	}
	if ptrName == "" {
		return nil, nil
	}

	// ex. "My Camera._hap._tcp.local." -> "My Camera"
	var name string
	if i := strings.Index(ptrName, "."+hapService); i > 0 {
		name = strings.ReplaceAll(ptrName[:i], `\ `, " ")
	}

	info := map[string]string{}
	for _, rr := range records {
		txt, ok := rr.(*dns.TXT)
		if !ok || txt.Hdr.Name != ptrName {
			continue
		}
		for _, s := range txt.Txt {
			k, v, _ := strings.Cut(s, "=")
			info[k] = v
		}
		break
	}

	category := info[txtCategory]
	if category != categoryCamera && category != categoryDoorbell {
		return nil, nil
	}

	categoryName := "camera"
	if category == categoryDoorbell {
		categoryName = "doorbell"
	}

	var port int
	for _, rr := range records {
		if srv, ok := rr.(*dns.SRV); ok && srv.Hdr.Name == ptrName {
			port = int(srv.Port)
			break
		}
	}

	return &MDNSResult{
		Name:     name,
		DeviceID: info[txtDeviceID],
		Model:    info[txtModel],
		Category: categoryName,
		Paired:   info[txtStatusFlags] == statusPaired,
		Port:     port,
	}, nil
}

func init() {
	dns.Id = func() uint16 { return 0 }
}
