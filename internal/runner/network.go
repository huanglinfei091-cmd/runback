package runner

import (
	"context"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/huanglinfei091-cmd/runback/internal/replay"
)

const managedNetwork = "runback-managed"

type NetworkError struct{ Detail string }

func (e *NetworkError) Error() string {
	return "Stage: EXECUTE\nCause: NETWORK\n" + e.Detail
}

func probeNetwork(ctx context.Context, p replay.Plan, image, network string) error {
	probeCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	probe := []string{"run", "--rm", "--network", network, "--label", "io.runback.managed=true", "--label", "io.runback.purpose=network-probe", "--entrypoint", "bash", image, "-c", "timeout 8 bash -c 'exec 3<>/dev/tcp/github.com/443'"}
	_, err := run(probeCtx, p, "docker", probe...)
	return err
}

// SelectNetwork probes the Docker default bridge without changing it. If the
// probe fails, exactly one existing or newly created RunBack-owned bridge is
// selected. No repository identity is available to this function.
func SelectNetwork(ctx context.Context, image string) (name, source string, err error) {
	home, e := os.MkdirTemp("", "runback-network-")
	if e != nil {
		return "", "", &NetworkError{"cannot prepare isolated network probe"}
	}
	defer os.RemoveAll(home)
	p := replay.Plan{Root: home, Home: home}
	if e = probeNetwork(ctx, p, image, "bridge"); e == nil {
		return "", "GENERIC_DEFAULT", nil
	}

	out, listErr := run(ctx, p, "docker", "network", "ls", "--filter", "label=io.runback.managed=true", "--format", "{{.Name}}")
	if listErr != nil {
		return "", "", &NetworkError{"default bridge probe failed and managed networks could not be listed"}
	}
	names := strings.Fields(out)
	sort.Strings(names)
	for _, candidate := range names {
		properties, inspectErr := run(ctx, p, "docker", "network", "inspect", candidate, "--format", `{{.Driver}}|{{.Internal}}|{{index .Labels "io.runback.managed"}}`)
		if inspectErr == nil && properties == "bridge|false|true" && probeNetwork(ctx, p, image, candidate) == nil {
			return candidate, "MANAGED_FALLBACK", nil
		}
	}

	created, createErr := run(ctx, p, "docker", "network", "create", "--driver", "bridge", "--label", "io.runback.managed=true", managedNetwork)
	if createErr != nil || strings.TrimSpace(created) == "" {
		return "", "", &NetworkError{"default bridge probe failed and one managed fallback could not be created"}
	}
	if probeNetwork(ctx, p, image, managedNetwork) != nil {
		return "", "", &NetworkError{"default bridge and the newly created managed fallback have no outbound HTTPS connectivity"}
	}
	return managedNetwork, "MANAGED_FALLBACK", nil
}
