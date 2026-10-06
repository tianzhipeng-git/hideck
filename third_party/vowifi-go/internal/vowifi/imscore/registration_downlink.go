package imscore

import "net"

// Evidence belongs to one protected connection and path generation. Comparing
// both prevents periodic REGISTER from discarding valid evidence while ensuring
// reconnects, security/path resets and registrar changes cannot reuse it.
type registrationDownlinkEvidence struct {
	conn       net.Conn
	checkpoint downlinkCheckpoint
}

// Caller holds mu. TS 33.203 permits downlink requests on the UE-originated TCP.
func (s *Service) registrationDownlinkReadyLocked() bool {
	return s.registrationTCP != nil && s.registrationTCPProtected &&
		s.registrationDownlink.conn == s.registrationTCP &&
		s.registrationDownlink.checkpoint.samePath(s.downlinkCheckpointLocked())
}
