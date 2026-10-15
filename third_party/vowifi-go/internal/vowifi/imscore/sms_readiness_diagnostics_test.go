package imscore

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net"
	"strings"
	"testing"
)

func TestSMSReadinessDiagnosticDoesNotChangeStateOrLeakIdentity(t *testing.T) {
	s := newProtectedKeepaliveTestService(t)
	conn, peer := net.Pipe()
	t.Cleanup(func() { conn.Close(); peer.Close() })
	s.registrationTCP, s.registrationTCPProtected = conn, true
	s.cfg.IMSI = "diagnostic-secret-imsi"
	s.cfg.SMSC = "diagnostic-secret-smsc"
	s.regSession.publicID = "sip:diagnostic-secret-identity@example.test"
	s.downlinkRequests = 1
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	before := s.SMSReadiness()
	s.logSMSReadinessDiagnostic("inbound_message", "inbound_path", s.inboundDiagnosticPath(conn))
	if after := s.SMSReadiness(); after != before {
		t.Fatalf("logging changed readiness: before=%+v after=%+v", before, after)
	}
	if strings.Contains(output.String(), "diagnostic-secret") {
		t.Fatal("identity leaked into diagnostic")
	}
	var fields map[string]any
	if err := json.Unmarshal(output.Bytes(), &fields); err != nil {
		t.Fatal(err)
	}
	if fields["inbound_path"] != "registration_tcp" || fields["sms_ready"] != false || fields["downlink_requests"] != float64(1) || fields["protected_push_required"] != true {
		t.Fatalf("missing diagnostic evidence: %v", fields)
	}
	if before.MOReady != true || before.Ready != false {
		t.Fatal("unexpected fixture")
	}
}

func TestInboundDiagnosticPathSeparatesLiveAndRetiredConnections(t *testing.T) {
	s := newProtectedKeepaliveTestService(t)
	registration, registrationPeer := net.Pipe()
	push, pushPeer := net.Pipe()
	t.Cleanup(func() { registration.Close(); registrationPeer.Close(); push.Close(); pushPeer.Close() })
	s.registrationTCP, s.registrationTCPProtected = registration, true
	if got := s.inboundDiagnosticPath(registration); got != "registration_tcp" {
		t.Fatal(got)
	}
	if got := s.inboundDiagnosticPath(&protectedUDPPeer{}); got != "protected_udp" {
		t.Fatal(got)
	}
	s.protectedConns[push] = struct{}{}
	if got := s.inboundDiagnosticPath(push); got != "port_s_tcp" {
		t.Fatal(got)
	}
	delete(s.protectedConns, push)
	if got := s.inboundDiagnosticPath(push); got != "retired_or_untracked_tcp" {
		t.Fatal(got)
	}
	if got := s.inboundDiagnosticPath(nil); got != "unattributed" {
		t.Fatal(got)
	}
	s.externalTransport = true
	if got := s.inboundDiagnosticPath(nil); got != "external" {
		t.Fatal(got)
	}
}
