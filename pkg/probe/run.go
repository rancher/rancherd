package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/rancher/system-agent/pkg/applyinator"
	"github.com/rancher/system-agent/pkg/prober"
	"github.com/sirupsen/logrus"
)

func RunProbes(ctx context.Context, planFile string, interval time.Duration) error {
	f, err := os.Open(planFile)
	if err != nil {
		return fmt.Errorf("opening plan %s: %w", planFile, err)
	}
	defer f.Close()

	plan := &applyinator.Plan{}
	if err := json.NewDecoder(f).Decode(plan); err != nil {
		return err
	}

	if len(plan.Probes) == 0 {
		logrus.Infof("No probes defined in %s", planFile)
		return nil
	}
	logrus.Infof("Running probes defined in %s", planFile)

	initial := true
	probeStatuses := map[string]prober.ProbeStatus{}
	for {
		newProbeStatuses := map[string]prober.ProbeStatus{}
		for k, v := range probeStatuses {
			newProbeStatuses[k] = v
		}
		prober.DoProbes(plan.Probes, newProbeStatuses, initial)

		allGood := true
		for probeName, probeStatus := range newProbeStatuses {
			if !probeStatus.Healthy {
				allGood = false
			}

			oldProbeStatus, ok := probeStatuses[probeName]
			if !ok || oldProbeStatus.Healthy != probeStatus.Healthy {
				if probeStatus.Healthy {
					logrus.Infof("Probe [%s] is healthy", probeName)
				} else {
					logrus.Infof("Probe [%s] is unhealthy", probeName)
				}
			}
		}

		if allGood {
			logrus.Info("All probes are healthy")
			break
		}

		probeStatuses = newProbeStatuses
		initial = false
		time.Sleep(interval)
	}

	return nil
}
