package groups

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/muthuishere/aos/internal/sys"
)

// coreAudioShim declares just enough of IAudioEndpointVolume for PowerShell to
// call it. Windows ships no command-line volume control, and this avoids making
// the CLI depend on a third-party tool like nircmd.
const coreAudioShim = `
Add-Type -ErrorAction SilentlyContinue @'
using System.Runtime.InteropServices;
[Guid("5CDF2C82-841E-4546-9722-0CF74078229A"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
interface IAudioEndpointVolume {
  // The vtable order is fixed by the COM interface, so every method before the
  // ones we call has to be declared even though it is unused. Getting the count
  // wrong does not fail to compile — it silently calls the neighbouring method.
  int RegisterControlChangeNotify(System.IntPtr notify);
  int UnregisterControlChangeNotify(System.IntPtr notify);
  int GetChannelCount(out int count);
  int SetMasterVolumeLevel(float level, System.Guid ctx);
  int SetMasterVolumeLevelScalar(float level, System.Guid ctx);
  int GetMasterVolumeLevel(out float level);
  int GetMasterVolumeLevelScalar(out float level);
  int SetChannelVolumeLevel(int channel, float level, System.Guid ctx);
  int SetChannelVolumeLevelScalar(int channel, float level, System.Guid ctx);
  int GetChannelVolumeLevel(int channel, out float level);
  int GetChannelVolumeLevelScalar(int channel, out float level);
  int SetMute(bool mute, System.Guid ctx);
  int GetMute(out bool mute);
}
[Guid("D666063F-1587-4E43-81F1-B948E807363F"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
interface IMMDevice { int Activate(ref System.Guid id, int ctx, System.IntPtr p, out IAudioEndpointVolume ep); }
[Guid("A95664D2-9614-4F35-A746-DE8DB63617E6"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
interface IMMDeviceEnumerator { int f(); int GetDefaultAudioEndpoint(int dataFlow, int role, out IMMDevice dev); }
[ComImport, Guid("BCDE0395-E52F-467C-8E3D-C4579291692E")] class MMDeviceEnumeratorComObject { }
public class Audio {
  static IAudioEndpointVolume Endpoint() {
    IMMDeviceEnumerator e = (IMMDeviceEnumerator)(new MMDeviceEnumeratorComObject());
    IMMDevice dev; e.GetDefaultAudioEndpoint(0, 1, out dev);
    System.Guid id = typeof(IAudioEndpointVolume).GUID;
    IAudioEndpointVolume ep; dev.Activate(ref id, 23, System.IntPtr.Zero, out ep);
    return ep;
  }
  public static float GetVolume() { float v; Endpoint().GetMasterVolumeLevelScalar(out v); return v; }
  public static void SetVolume(float v) { Endpoint().SetMasterVolumeLevelScalar(v, System.Guid.Empty); }
  public static bool GetMute() { bool m; Endpoint().GetMute(out m); return m; }
  public static void SetMute(bool m) { Endpoint().SetMute(m, System.Guid.Empty); }
}
'@
`

func getVolume() (int, error) {
	out, err := sys.PowerShell(coreAudioShim + "[Audio]::GetVolume()")
	if err != nil {
		return 0, err
	}
	scalar, err := strconv.ParseFloat(strings.TrimSpace(out), 64)
	if err != nil {
		return 0, fmt.Errorf("unexpected volume reading %q", out)
	}
	return int(math.Round(scalar * 100)), nil
}

func setVolume(level int) error {
	_, err := sys.PowerShell(coreAudioShim + fmt.Sprintf("[Audio]::SetVolume(%f)", float64(level)/100))
	return err
}

func getMute() (bool, error) {
	out, err := sys.PowerShell(coreAudioShim + "[Audio]::GetMute()")
	if err != nil {
		return false, err
	}
	return strings.EqualFold(strings.TrimSpace(out), "True"), nil
}

func setMute(muted bool) error {
	_, err := sys.PowerShell(coreAudioShim + fmt.Sprintf("[Audio]::SetMute($%t)", muted))
	return err
}
