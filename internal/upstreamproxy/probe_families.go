package upstreamproxy

import (
	"context"
	"errors"
	"strings"
	"time"
)

// ProbeSOCKS5 checks each destination address family in its own UDP association.
// As before, a valid response from any target is sufficient to pass.
func ProbeSOCKS5(ctx context.Context, cfg ProbeConfig) (ProbeResult, error) {
	startedAt := time.Now()
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	ctx, cancel := context.WithDeadline(ctx, probeDeadline(ctx, cfg.Timeout))
	defer cancel()
	requests, err := prepareUDPProbeRequests(ctx, cfg.UDPProbeTargets)
	if strings.TrimSpace(cfg.ProxyAddr) == "" {
		err = errors.New("empty proxy addr")
	}
	if err != nil {
		return finalizeProbeResult(ProbeResult{
			ProxyAddr: strings.TrimSpace(cfg.ProxyAddr), Stage: ProbeStageTCPConnect, Error: err.Error(),
		}, startedAt), err
	}
	var families [2][]udpProbeRequest
	for _, request := range requests {
		family := 0
		if request.target.IP.To4() == nil {
			family = 1
		}
		families[family] = append(families[family], request)
	}
	type outcome struct {
		result ProbeResult
		err    error
	}
	results := make(chan outcome, 2)
	pending := 0
	for _, family := range families {
		if len(family) == 0 {
			continue
		}
		pending++
		go func(requests []udpProbeRequest) {
			result, err := probeSOCKS5Session(ctx, cfg, requests)
			results <- outcome{result, err}
		}(family)
	}
	var best outcome
	for i := 0; i < pending; i++ {
		current := <-results
		if current.err == nil && !best.result.OK() {
			best = current
			cancel()
		} else if !best.result.OK() && (i == 0 || probeProgress(current.result) > probeProgress(best.result)) {
			best = current
		}
	}
	return finalizeProbeResult(best.result, startedAt), best.err
}

func probeProgress(result ProbeResult) int {
	if result.UDPAssociateOK {
		return 3
	}
	if result.HandshakeOK {
		return 2
	}
	if result.Reachable {
		return 1
	}
	return 0
}
