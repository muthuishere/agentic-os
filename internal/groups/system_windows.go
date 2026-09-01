package groups

import "github.com/muthuishere/aos/internal/sys"

func systemLock() error {
	_, err := sys.Output("rundll32.exe", "user32.dll,LockWorkStation")
	return err
}

func systemSleep() error {
	// The third argument disables wake timers; hibernation stays off so this is
	// a true suspend-to-RAM on machines with hibernation disabled.
	_, err := sys.Output("rundll32.exe", "powrprof.dll,SetSuspendState", "0,1,0")
	return err
}

func systemRestart() error {
	_, err := sys.Output("shutdown", "/r", "/t", "0")
	return err
}

func systemShutdown() error {
	_, err := sys.Output("shutdown", "/s", "/t", "0")
	return err
}

func systemLogout() error {
	_, err := sys.Output("shutdown", "/l")
	return err
}

func systemInfo() (map[string]string, error) {
	facts := map[string]string{"platform": "windows"}
	out, err := sys.PowerShell(`
$os = Get-CimInstance Win32_OperatingSystem
$cs = Get-CimInstance Win32_ComputerSystem
$cpu = Get-CimInstance Win32_Processor | Select-Object -First 1
"os=" + $os.Caption
"os_version=" + $os.Version
"build=" + $os.BuildNumber
"arch=" + $env:PROCESSOR_ARCHITECTURE
"host=" + $env:COMPUTERNAME
"cpu=" + $cpu.Name
"model=" + $cs.Manufacturer + " " + $cs.Model
`)
	if err != nil {
		return facts, nil
	}
	mergeKeyValues(facts, out)
	return facts, nil
}
