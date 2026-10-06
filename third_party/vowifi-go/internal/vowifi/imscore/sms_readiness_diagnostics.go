package imscore

import (
	"net"

	"github.com/iniwex5/vowifi-go/internal/vowifi/logging"
)

// Only event-driven callers use this diagnostic; polling SMSReadiness stays quiet.
// Do not add raw SIP, message bodies, identities, Call-ID or error strings here.
// Readiness and path snapshots are sampled separately and can straddle recovery.
func (s *Service) logSMSReadinessDiagnostic(event string, extra ...any) {
	readiness := s.SMSReadiness()
	s.mu.RLock()
	checkpoint := s.downlinkCheckpointLocked()
	baseReceiver := s.smsReceiverReady
	pushRequired := s.protectedSMSPushRequiredLocked()
	registrationProtected := s.registrationTCPProtected
	registrationConnected := s.registrationTCP != nil
	registrationProven := s.registrationDownlinkReadyLocked()
	udpProven := s.udpDownlinkProven.Load()
	portSConnected := s.portSPushReady.Load()
	s.mu.RUnlock()
	portS := s.capturePortSSession()
	fields := []any{
		"device", s.DeviceID(), "event", event,
		"registration_generation", s.registrationGeneration.Load(),
		"pcscf", checkpoint.registrar, "downlink_generation", checkpoint.generation,
		"downlink_requests", checkpoint.requests,
		"registered", readiness.Registered, "profile_ready", readiness.ProfileReady,
		"transport_ready", readiness.TransportReady, "base_receiver_ready", baseReceiver,
		"protected_push_required", pushRequired, "port_s_connected", portSConnected,
		"registration_tcp_connected", registrationConnected,
		"registration_tcp_protected", registrationProtected,
		"registration_downlink_proven", registrationProven,
		"udp_downlink_proven", udpProven, "port_s_generation", portS.generation,
		"port_s_last_close_kind", portS.lastCloseKind,
		"receiver_ready", readiness.ReceiverReady, "sms_ready", readiness.Ready,
		"sms_health_ready", readiness.HealthReady, "sms_mo_ready", readiness.MOReady,
		"reason", readiness.Reason,
	}
	logging.Info("IMS SMS readiness diagnostic", append(fields, extra...)...)
}

func (s *Service) inboundDiagnosticPath(peer net.Conn) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := peer.(*protectedUDPPeer); ok {
		return "protected_udp"
	}
	if peer != nil && peer == s.registrationTCP {
		return "registration_tcp"
	}
	if peer == nil {
		if s.externalTransport {
			return "external"
		}
		return "unattributed"
	}
	s.protectedConnMu.Lock()
	_, tracked := s.protectedConns[peer]
	s.protectedConnMu.Unlock()
	if tracked {
		return "port_s_tcp"
	}
	return "retired_or_untracked_tcp"
}
