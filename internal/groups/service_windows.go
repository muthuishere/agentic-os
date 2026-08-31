package groups

import (
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/muthuishere/agentic-os/internal/sys"
)

// Windows services proper (sc.exe / New-Service) require an elevated shell, so
// a per-user `service create` would prompt for admin every time. A Scheduled
// Task needs no elevation, runs as the logged-in user, and has the ONLOGON
// trigger this group wants — so schtasks is the backend.

// taskName is the scheduled task's path. The namespace prefix is what keeps
// `list` and `remove` confined to tasks aos created.
func taskName(label string) string { return `\` + label }

func schtasks(args ...string) (string, error) {
	// The Task Scheduler service is the parent of anything a task starts, so
	// these calls never inherit our stdio to a long-lived child.
	return sys.Output("schtasks", args...)
}

func serviceInstall(spec serviceSpec) ([]string, error) {
	name := taskName(spec.Label)
	if _, err := schtasks("/Create", "/TN", name,
		"/TR", schtasksCommand(spec.Command), "/SC", "ONLOGON", "/F"); err != nil {
		return nil, err
	}
	notes := []string{"scheduled task " + name}
	if !spec.Autostart {
		// A scheduled task's only "do not start at logon" state is disabled;
		// `schtasks /Run` still starts a disabled task on demand, so `start`
		// keeps working without turning the logon trigger back on.
		if _, err := schtasks("/Change", "/TN", name, "/DISABLE"); err != nil {
			return nil, err
		}
	}
	return notes, nil
}

func serviceUninstall(label string) error {
	if info, err := serviceQuery(label); err == nil && info.State == serviceNotInstalled {
		return fmt.Errorf("service %q is not installed", label)
	}
	_, _ = schtasks("/End", "/TN", taskName(label))
	_, err := schtasks("/Delete", "/TN", taskName(label), "/F")
	return err
}

func serviceStartOne(label string) error {
	_, err := schtasks("/Run", "/TN", taskName(label))
	return err
}

func serviceStopOne(label string) error {
	_, err := schtasks("/End", "/TN", taskName(label))
	return err
}

func serviceQuery(label string) (serviceInfo, error) {
	// /V widens the CSV to the verbose column set; status is read by position
	// because the column headers are localized and the values are not stable
	// enough to match by name.
	out, err := schtasks("/Query", "/TN", taskName(label), "/FO", "CSV", "/NH", "/V")
	if err != nil {
		return serviceInfo{State: serviceNotInstalled}, nil
	}
	records, err := csv.NewReader(strings.NewReader(out)).ReadAll()
	if err != nil || len(records) == 0 || len(records[0]) < 4 {
		return serviceInfo{State: serviceStopped}, nil
	}
	status := strings.TrimSpace(records[0][3])
	if strings.EqualFold(status, "Running") {
		return serviceInfo{State: serviceRunning}, nil
	}
	return serviceInfo{State: serviceStopped, Detail: strings.ToLower(status)}, nil
}

func serviceLabels() ([]string, error) {
	out, err := schtasks("/Query", "/FO", "CSV", "/NH")
	if err != nil {
		return nil, err
	}
	reader := csv.NewReader(strings.NewReader(out))
	// Task listings vary in width across Windows versions; only the first
	// column matters here.
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	var labels []string
	for _, record := range records {
		if len(record) == 0 {
			continue
		}
		label := strings.TrimPrefix(strings.TrimSpace(record[0]), `\`)
		if _, ok := serviceShortName(label); ok {
			labels = append(labels, label)
		}
	}
	return labels, nil
}
