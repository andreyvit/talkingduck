package telemetry

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"net"
	"time"
)

func ping(parent context.Context, target string) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(parent, 900*time.Millisecond)
	defer cancel()
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, target)
	if err != nil {
		return 0, err
	}
	if len(ips) == 0 {
		return 0, errors.New("no address")
	}
	ip := ips[0]
	network, bind, proto := "udp4", "0.0.0.0", 1
	var typ icmp.Type = ipv4.ICMPTypeEcho
	var reply icmp.Type = ipv4.ICMPTypeEchoReply
	if ip.IP.To4() == nil {
		network, bind, proto = "udp6", "::", 58
		typ = ipv6.ICMPTypeEchoRequest
		reply = ipv6.ICMPTypeEchoReply
	}
	c, err := icmp.ListenPacket(network, bind)
	if err != nil {
		return 0, err
	}
	defer c.Close()
	stop := context.AfterFunc(ctx, func() { c.Close() })
	defer stop()
	deadline, _ := ctx.Deadline()
	if err = c.SetDeadline(deadline); err != nil {
		return 0, err
	}
	payload := make([]byte, 16)
	if _, err = rand.Read(payload); err != nil {
		return 0, err
	}
	b, err := (&icmp.Message{Type: typ, Body: &icmp.Echo{ID: 1, Seq: 1, Data: payload}}).Marshal(nil)
	if err != nil {
		return 0, err
	}
	start := time.Now()
	if _, err = c.WriteTo(b, &net.UDPAddr{IP: ip.IP, Zone: ip.Zone}); err != nil {
		return 0, err
	}
	buf := make([]byte, 1500)
	for {
		n, peer, err := c.ReadFrom(buf)
		if err != nil {
			return 0, err
		}
		msg, err := icmp.ParseMessage(proto, buf[:n])
		if err != nil || msg.Type != reply {
			continue
		}
		echo, ok := msg.Body.(*icmp.Echo)
		if !ok || echo.Seq != 1 || !bytes.Equal(echo.Data, payload) {
			continue
		}
		addr, ok := peer.(*net.UDPAddr)
		if ok && !addr.IP.Equal(ip.IP) {
			continue
		}
		return time.Since(start), nil
	}
}
