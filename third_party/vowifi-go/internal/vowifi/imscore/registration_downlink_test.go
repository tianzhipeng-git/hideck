package imscore

import (
	"github.com/iniwex5/vowifi-go/internal/vowifi/events"
	"net"
	"testing"
	"time"
)

func TestRegistrationDownlinkReadinessFollowsCurrentPath(t *testing.T) {
	for _, change := range []string{"connection", "generation", "registrar", "closed"} {
		t.Run(change, func(t *testing.T) {
			s := newProtectedKeepaliveTestService(t)
			conn, peer := net.Pipe()
			next, nextPeer := net.Pipe()
			t.Cleanup(func() { conn.Close(); peer.Close(); next.Close(); nextPeer.Close() })
			s.registrationTCP, s.registrationTCPProtected = conn, true
			if got := s.SMSReadiness(); got.Ready || got.Reason != "IMS SMS downlink is awaiting verification" {
				t.Fatalf("initial: %+v", got)
			}
			callbacks := 0
			s.SetOnSMSReadinessChanged(func(r SMSReadiness) {
				if r.Ready {
					callbacks++
				}
			})
			raw := optionsRequest("registration-downlink")
			message, err := parseSIPMessage(raw)
			if err != nil {
				t.Fatal(err)
			}
			if err = s.dispatchInboundSIPMessageWithPeer(message, raw, func(string) error { return nil }, conn); err != nil {
				t.Fatal(err)
			}
			if got := s.SMSReadiness(); !got.Ready || !got.HealthReady || callbacks != 1 {
				t.Fatalf("after inbound: %+v callbacks=%d", got, callbacks)
			}
			old := s.captureDownlinkCheckpoint()
			switch change {
			case "connection":
				s.registrationTCP = next
			case "generation":
				s.resetPortSRecoveryKnowledge()
			case "registrar":
				s.registrar = "replacement.example:5060"
			case "closed":
				s.registrationTCP = nil
			}
			if s.SMSReadiness().Ready {
				t.Fatal("old evidence survived replacement")
			}
			s.recordCurrentDownlinkRequest(conn, old)
			if s.SMSReadiness().Ready {
				t.Fatal("late request validated replacement")
			}
		})
	}
}

func TestRegistrationDownlinkDoesNotAcceptFailedRequests(t *testing.T) {
	s := newProtectedKeepaliveTestService(t)
	conn, peer := net.Pipe()
	t.Cleanup(func() { conn.Close(); peer.Close() })
	s.registrationTCP, s.registrationTCPProtected = conn, true
	raw := optionsRequest("failed-reply")
	message, err := parseSIPMessage(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.dispatchInboundSIPMessageWithPeer(message, raw, func(string) error { return net.ErrClosed }, conn); err == nil {
		t.Fatal("expected reply failure")
	}
	if s.SMSReadiness().Ready {
		t.Fatal("failed handler proved readiness")
	}
}

func TestRegistrationDownlinkSMSDeliveryUpdatesReadiness(t *testing.T) {
	s, subscriber, _ := newInboundSMSTestService(t)
	conn, peer := net.Pipe()
	t.Cleanup(func() { conn.Close(); peer.Close() })
	s.externalTransport = false
	s.smsReceiverReady = true
	s.registrationTCP, s.registrationTCPProtected = conn, true
	s.protectedClientPort, s.protectedServerPort = 16082, 16083
	s.registrationRemote = &net.UDPAddr{IP: net.IPv4(10, 0, 0, 1), Port: 5060}
	ready := 0
	s.SetOnSMSReadinessChanged(func(r SMSReadiness) {
		if r.Ready {
			ready++
		}
	})
	writes := make(chan string, 2)
	readErr := make(chan error, 1)
	go (peerSMSCapture{service: s, peer: peer, writes: writes, readErr: readErr}).run()
	raw := inboundSMSRequest(t, imsSMSContentType, inboundRPData(t, 0x35, "+447700900123", "readiness"))
	message, err := parseSIPMessage(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.dispatchInboundSIPMessageWithPeer(message, raw, func(response string) error { return s.writeSIPStream(conn, response) }, conn); err != nil {
		t.Fatal(err)
	}
	waitForPeerSIPWrite(t, writes, readErr)
	waitForPeerSIPWrite(t, writes, readErr)
	select {
	case event := <-subscriber.events:
		if _, ok := event.(*events.EventSMSReceived); !ok {
			t.Fatalf("unexpected event %T", event)
		}
	case <-time.After(time.Second):
		t.Fatal("SMS not delivered")
	}
	if !s.SMSReadiness().Ready || ready != 1 {
		t.Fatalf("SMS delivered but readiness not updated: %+v callbacks=%d", s.SMSReadiness(), ready)
	}
}
