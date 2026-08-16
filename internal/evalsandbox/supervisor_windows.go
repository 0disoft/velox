//go:build windows

package evalsandbox

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/0disoft/velox/internal/buildinfo"
	"golang.org/x/sys/windows"
)

const procThreadAttributeSecurityCapabilities = 0x00020009

var (
	userenvDLL                = windows.NewLazySystemDLL("userenv.dll")
	createAppContainerProfile = userenvDLL.NewProc("CreateAppContainerProfile")
	deleteAppContainerProfile = userenvDLL.NewProc("DeleteAppContainerProfile")
	isProcessInJob            = windows.NewLazySystemDLL("kernel32.dll").NewProc("IsProcessInJob")
)

type securityCapabilities struct {
	AppContainerSID *windows.SID
	Capabilities    *windows.SIDAndAttributes
	CapabilityCount uint32
	Reserved        uint32
}

func Run(config Config) (Receipt, error) {
	prepared, err := prepare(config)
	if err != nil {
		return Receipt{}, err
	}
	privateEnvironmentCleanupNeeded := true
	evaluationStateCleanupNeeded := true
	defer func() {
		if privateEnvironmentCleanupNeeded {
			_ = os.RemoveAll(prepared.PrivateEnvironmentRoot)
		}
		if evaluationStateCleanupNeeded {
			_ = os.RemoveAll(filepath.Dir(prepared.StateDatabasePath))
		}
	}()
	supervisorPath, err := os.Executable()
	if err != nil {
		return Receipt{}, fmt.Errorf("resolve supervisor executable: %w", err)
	}
	supervisorSHA, err := fileDigest(supervisorPath)
	if err != nil {
		return Receipt{}, fmt.Errorf("digest supervisor executable: %w", err)
	}
	commandSHA, err := commandDigest(prepared.Command)
	if err != nil {
		return Receipt{}, fmt.Errorf("digest sandbox command: %w", err)
	}

	profileName, err := randomProfileName()
	if err != nil {
		return Receipt{}, err
	}
	internetClient, err := windows.StringToSid("S-1-15-3-1")
	if err != nil {
		return Receipt{}, fmt.Errorf("construct internetClient capability: %w", err)
	}
	capabilities := []windows.SIDAndAttributes{{Sid: internetClient, Attributes: windows.SE_GROUP_ENABLED}}
	appContainerSID, err := createProfile(profileName, capabilities)
	if err != nil {
		return Receipt{}, err
	}
	defer windows.FreeSid(appContainerSID) //nolint:errcheck -- process cleanup reports profile and ACL failures

	applied := make([]preparedGrant, 0, len(prepared.Grants))
	cleanup := func() error {
		var cleanupErrors []error
		for index := len(applied) - 1; index >= 0; index-- {
			if revokeErr := updatePathAccess(applied[index], appContainerSID, windows.REVOKE_ACCESS); revokeErr != nil {
				cleanupErrors = append(cleanupErrors, fmt.Errorf("revoke %s grant: %w", applied[index].Role, revokeErr))
			}
		}
		if deleteErr := deleteProfile(profileName); deleteErr != nil {
			cleanupErrors = append(cleanupErrors, deleteErr)
		}
		if removeErr := os.RemoveAll(prepared.PrivateEnvironmentRoot); removeErr != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("remove private sandbox environment: %w", removeErr))
		} else {
			privateEnvironmentCleanupNeeded = false
		}
		if removeErr := os.RemoveAll(filepath.Dir(prepared.StateDatabasePath)); removeErr != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("remove isolated Hermes state: %w", removeErr))
		} else {
			evaluationStateCleanupNeeded = false
		}
		return errors.Join(cleanupErrors...)
	}

	for _, grant := range prepared.Grants {
		if err = updatePathAccess(grant, appContainerSID, windows.GRANT_ACCESS); err != nil {
			cleanupErr := cleanup()
			return Receipt{}, errors.Join(fmt.Errorf("apply %s grant: %w", grant.Role, err), cleanupErr)
		}
		applied = append(applied, grant)
	}

	started := time.Now().UTC()
	exitCode, timedOut, runErr := launchContained(prepared, appContainerSID, capabilities)
	finished := time.Now().UTC()
	if runErr != nil || timedOut || exitCode != 0 {
		cleanupErr := cleanup()
		if timedOut {
			runErr = errors.Join(runErr, fmt.Errorf("sandbox command exceeded %s", prepared.Timeout))
		} else if exitCode != 0 {
			runErr = errors.Join(runErr, fmt.Errorf("sandbox command exited with code %d", exitCode))
		}
		return Receipt{}, errors.Join(runErr, cleanupErr)
	}
	stateDatabaseSHA, exportErr := exportStateDatabase(prepared.StateDatabasePath, prepared.StateDatabaseExportPath)
	cleanupErr := cleanup()
	if exportErr != nil || cleanupErr != nil {
		_ = os.Remove(prepared.StateDatabaseExportPath)
		return Receipt{}, errors.Join(exportErr, cleanupErr)
	}

	receipt := Receipt{
		SchemaVersion: ReceiptVersion,
		TrialID:       prepared.TrialID,
		SeriesID:      prepared.SeriesID,
		Sequence:      prepared.Sequence,
		Policy: Policy{
			SchemaVersion:      PolicyVersion,
			Platform:           "windows",
			FilesystemBoundary: "appcontainer-explicit-acl",
			ProcessBoundary:    "job-object-no-breakaway",
			NetworkCapability:  "internet-client",
		},
		Supervisor:          Supervisor{Version: buildinfo.Version, SHA256: supervisorSHA},
		CommandSHA256:       commandSHA,
		EnvironmentSHA256:   prepared.EnvironmentSHA256,
		PromptSHA256:        prepared.PromptSHA256,
		StateDatabaseSHA256: stateDatabaseSHA,
		StartedAtUTC:        started.Format(time.RFC3339Nano),
		FinishedAtUTC:       finished.Format(time.RFC3339Nano),
		ExitCode:            exitCode,
		TimedOut:            false,
		Containment: Containment{
			FilesystemEnforced:  true,
			ProcessTreeEnforced: true,
			CleanupCompleted:    true,
		},
		Grants: receiptGrants(prepared.Grants),
	}
	if err := WriteReceiptExclusive(prepared.ReceiptPath, receipt); err != nil {
		_ = os.Remove(prepared.StateDatabaseExportPath)
		return Receipt{}, fmt.Errorf("write sandbox receipt: %w", err)
	}
	return receipt, nil
}

func randomProfileName() (string, error) {
	value := make([]byte, 12)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("create profile identity: %w", err)
	}
	return "velox.eval." + hex.EncodeToString(value), nil
}

func createProfile(name string, capabilities []windows.SIDAndAttributes) (*windows.SID, error) {
	nameUTF16, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}
	displayUTF16, _ := windows.UTF16PtrFromString("Velox evaluation sandbox")
	descriptionUTF16, _ := windows.UTF16PtrFromString("Ephemeral AppContainer for one Velox clean-room evaluation")
	var sid *windows.SID
	result, _, _ := createAppContainerProfile.Call(
		uintptr(unsafe.Pointer(nameUTF16)),
		uintptr(unsafe.Pointer(displayUTF16)),
		uintptr(unsafe.Pointer(descriptionUTF16)),
		uintptr(unsafe.Pointer(&capabilities[0])),
		uintptr(len(capabilities)),
		uintptr(unsafe.Pointer(&sid)),
	)
	if hr := int32(uint32(result)); hr < 0 {
		return nil, fmt.Errorf("create AppContainer profile: HRESULT 0x%08x", uint32(hr))
	}
	if sid == nil {
		return nil, fmt.Errorf("create AppContainer profile returned no SID")
	}
	return sid, nil
}

func deleteProfile(name string) error {
	nameUTF16, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return err
	}
	result, _, _ := deleteAppContainerProfile.Call(uintptr(unsafe.Pointer(nameUTF16)))
	if hr := int32(uint32(result)); hr < 0 {
		return fmt.Errorf("delete AppContainer profile: HRESULT 0x%08x", uint32(hr))
	}
	return nil
}

func updatePathAccess(grant preparedGrant, sid *windows.SID, mode windows.ACCESS_MODE) error {
	descriptor, err := windows.GetNamedSecurityInfo(grant.Path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return err
	}
	currentACL, _, err := descriptor.DACL()
	if err != nil {
		return err
	}
	permissions := windows.ACCESS_MASK(windows.GENERIC_READ | windows.GENERIC_EXECUTE)
	if grant.Rights == "read-write-execute" {
		permissions = windows.GENERIC_ALL
	}
	entry := windows.EXPLICIT_ACCESS{
		AccessPermissions: permissions,
		AccessMode:        mode,
		Inheritance:       windows.SUB_CONTAINERS_AND_OBJECTS_INHERIT,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  windows.TRUSTEE_IS_WELL_KNOWN_GROUP,
			TrusteeValue: windows.TrusteeValueFromSID(sid),
		},
	}
	updatedACL, err := windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{entry}, currentACL)
	if err != nil {
		return err
	}
	return windows.SetNamedSecurityInfo(grant.Path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION, nil, nil, updatedACL, nil)
}

func launchContained(config preparedConfig, appContainerSID *windows.SID, capabilities []windows.SIDAndAttributes) (uint32, bool, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, false, fmt.Errorf("create job object: %w", err)
	}
	defer windows.CloseHandle(job) //nolint:errcheck -- no receipt is emitted until the process tree is terminated
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err = windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); err != nil {
		return 0, false, fmt.Errorf("configure job object: %w", err)
	}

	attributeList, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		return 0, false, fmt.Errorf("create process attribute list: %w", err)
	}
	defer attributeList.Delete()
	security := securityCapabilities{
		AppContainerSID: appContainerSID,
		Capabilities:    &capabilities[0],
		CapabilityCount: uint32(len(capabilities)),
	}
	if err = attributeList.Update(procThreadAttributeSecurityCapabilities, unsafe.Pointer(&security), unsafe.Sizeof(security)); err != nil {
		return 0, false, fmt.Errorf("configure AppContainer process attribute: %w", err)
	}

	application, err := windows.UTF16PtrFromString(config.Executable)
	if err != nil {
		return 0, false, err
	}
	commandLine, err := windows.UTF16PtrFromString(windows.ComposeCommandLine(config.Command))
	if err != nil {
		return 0, false, err
	}
	workingDirectory, err := windows.UTF16PtrFromString(config.TrialRoot)
	if err != nil {
		return 0, false, err
	}
	startup := windows.StartupInfoEx{}
	startup.StartupInfo.Cb = uint32(unsafe.Sizeof(startup))
	startup.ProcThreadAttributeList = attributeList.List()
	process := windows.ProcessInformation{}
	environment, err := environmentBlock(config.Environment)
	if err != nil {
		return 0, false, err
	}
	flags := uint32(windows.CREATE_SUSPENDED | windows.CREATE_UNICODE_ENVIRONMENT | windows.EXTENDED_STARTUPINFO_PRESENT)
	if err = windows.CreateProcess(application, commandLine, nil, nil, false, flags, &environment[0], workingDirectory, &startup.StartupInfo, &process); err != nil {
		return 0, false, fmt.Errorf("create AppContainer process: %w", err)
	}
	defer windows.CloseHandle(process.Process) //nolint:errcheck -- handle close cannot weaken an already completed run
	defer windows.CloseHandle(process.Thread)  //nolint:errcheck -- handle close cannot weaken an already completed run
	if err = windows.AssignProcessToJobObject(job, process.Process); err != nil {
		_ = windows.TerminateProcess(process.Process, 125)
		return 0, false, fmt.Errorf("assign process to job object: %w", err)
	}
	contained, containmentErr := processBelongsToJob(process.Process, job)
	if containmentErr != nil || !contained {
		_ = windows.TerminateProcess(process.Process, 125)
		return 0, false, errors.Join(fmt.Errorf("verify process job containment: contained=%t", contained), containmentErr)
	}
	if _, err = windows.ResumeThread(process.Thread); err != nil {
		_ = windows.TerminateJobObject(job, 125)
		return 0, false, fmt.Errorf("resume sandbox process: %w", err)
	}

	waitMilliseconds := uint32(config.Timeout / time.Millisecond)
	event, err := windows.WaitForSingleObject(process.Process, waitMilliseconds)
	if err != nil {
		_ = windows.TerminateJobObject(job, 125)
		return 0, false, fmt.Errorf("wait for sandbox process: %w", err)
	}
	timedOut := event == uint32(windows.WAIT_TIMEOUT)
	if event != windows.WAIT_OBJECT_0 && !timedOut {
		_ = windows.TerminateJobObject(job, 125)
		return 0, false, fmt.Errorf("unexpected sandbox wait result 0x%x", event)
	}
	if timedOut {
		if err = windows.TerminateJobObject(job, 124); err != nil {
			return 0, true, fmt.Errorf("terminate timed-out process tree: %w", err)
		}
		_, _ = windows.WaitForSingleObject(process.Process, 5000)
	}
	var exitCode uint32
	if err = windows.GetExitCodeProcess(process.Process, &exitCode); err != nil {
		return 0, timedOut, fmt.Errorf("read sandbox exit code: %w", err)
	}
	if err = windows.TerminateJobObject(job, exitCode); err != nil && !errors.Is(err, syscall.ERROR_ACCESS_DENIED) {
		return 0, timedOut, fmt.Errorf("terminate residual process tree: %w", err)
	}
	return exitCode, timedOut, nil
}

func processBelongsToJob(process, job windows.Handle) (bool, error) {
	var result int32
	callResult, _, callErr := isProcessInJob.Call(uintptr(process), uintptr(job), uintptr(unsafe.Pointer(&result)))
	if callResult == 0 {
		return false, callErr
	}
	return result != 0, nil
}

func environmentBlock(values []string) ([]uint16, error) {
	result := make([]uint16, 0, 512)
	for _, value := range values {
		encoded, err := windows.UTF16FromString(value)
		if err != nil {
			return nil, fmt.Errorf("encode sandbox environment: %w", err)
		}
		result = append(result, encoded...)
	}
	result = append(result, 0)
	return result, nil
}

func SafeSummary(receipt Receipt) string {
	return strings.Join([]string{receipt.SchemaVersion, receipt.TrialID, receipt.Supervisor.Version}, " ")
}
