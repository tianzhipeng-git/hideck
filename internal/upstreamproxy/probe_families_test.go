package upstreamproxy

import (
	"context"
	"net"
	"testing"
	"time"
)

// The relay drops associations mixing destination families, as observed with
// the subscription proxy. Either family alone must still be able to pass.
func TestProbeSOCKS5SeparatesDestinationFamilies(t *testing.T) {
	for _, supported := range []string{"IPv4", "IPv6"} {
		t.Run(supported, func(t *testing.T) {
			relay, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
			if err != nil {
				t.Fatal(err)
			}
			defer relay.Close()
			addr := startProbeServer(t, func(conn net.Conn) {
				defer conn.Close()
				readBytes(t, conn, 3)
				writeBytes(t, conn, []byte{5, 0})
				readBytes(t, conn, 10)
				writeBytes(t, conn, buildUDPAssociateResponse(relay.LocalAddr().(*net.UDPAddr)))
			})
			checked := make(chan bool, 1)
			go func() {
				_ = relay.SetDeadline(time.Now().Add(time.Second))
				families := make(map[string]string)
				var reply []byte
				var recipient *net.UDPAddr
				mixed := false
				for i := 0; i < 6; i++ {
					buf := make([]byte, 4096)
					n, source, err := relay.ReadFromUDP(buf)
					if err != nil {
						checked <- false
						return
					}
					target, payload, err := decodeSOCKS5UDPDatagram(buf[:n])
					if err != nil {
						checked <- false
						return
					}
					family := "IPv6"
					if target.IP.To4() != nil {
						family = "IPv4"
					}
					if previous, ok := families[source.String()]; ok && previous != family {
						mixed = true
					}
					families[source.String()] = family
					if family == supported {
						payload[2] |= 0x80
						reply = buildSOCKS5UDPDatagram(target, payload)
						recipient = source
					}
				}
				valid := !mixed && len(families) == 2 && recipient != nil
				if valid {
					_, _ = relay.WriteToUDP(reply, recipient)
				}
				checked <- valid
			}()
			result, err := ProbeSOCKS5(context.Background(), ProbeConfig{ProxyAddr: addr, Timeout: 2 * time.Second})
			if valid := <-checked; !valid {
				t.Fatal("probe mixed destination families in one association")
			}
			if err != nil || !result.OK() {
				t.Fatalf("%s-only relay should pass: %+v, %v", supported, result, err)
			}
			target, err := net.ResolveUDPAddr("udp", result.UDPProbeTarget)
			if err != nil {
				t.Fatal(err)
			}
			if (target.IP.To4() != nil) != (supported == "IPv4") {
				t.Fatalf("unexpected successful target %s", target)
			}
		})
	}
}
