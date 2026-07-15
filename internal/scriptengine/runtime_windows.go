//go:build windows

package scriptengine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	goruntime "runtime"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	appContainerProfileName = "Yotta.Script.Worker.3_1"
	workerStartupAllowance  = 5 * time.Second
	workerTerminationCode   = 0x59303131

	procThreadAttributeSecurityCapabilities          = 0x00020009
	procThreadAttributeJobList                       = 0x0002000d
	procThreadAttributeChildProcessPolicy            = 0x0002000e
	procThreadAttributeAllApplicationPackages        = 0x0002000f
	processCreationChildProcessRestricted     uint32 = 0x00000001
	processCreationAllPackagesOptOut          uint32 = 0x00000001
	hresultAlreadyExists                             = 0x800700b7
	maxWorkerExecutableBytes                         = 512 << 20
)

var (
	userenvDLL                = windows.NewLazySystemDLL("userenv.dll")
	createAppContainerProfile = userenvDLL.NewProc("CreateAppContainerProfile")
	deriveAppContainerSID     = userenvDLL.NewProc("DeriveAppContainerSidFromAppContainerName")
	getAppContainerFolderPath = userenvDLL.NewProc("GetAppContainerFolderPath")
)

type securityCapabilities struct {
	AppContainerSID *windows.SID
	Capabilities    unsafe.Pointer
	CapabilityCount uint32
	Reserved        uint32
}

type windowsRuntime struct {
	options RuntimeOptions
	mu      sync.Mutex
}

type processResult struct {
	exitCode uint32
	err      error
}

type responseResult struct {
	response Response
	err      error
}

func newPlatformRuntime(options RuntimeOptions) Runtime {
	return &windowsRuntime{options: options}
}

func (runtime *windowsRuntime) Execute(ctx context.Context, request Request) (Response, error) {
	if err := request.Validate(); err != nil {
		return Response{}, err
	}
	if err := ctx.Err(); err != nil {
		return Response{}, err
	}
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return Response{}, err
	}
	response, failureCode, err := runtime.execute(ctx, request)
	if err == nil {
		return response, nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return Response{}, ctxErr
	}
	message := "script worker failed"
	if failureCode == CodeIsolationUnavailable {
		message = "script isolation is unavailable on this host"
	} else if failureCode == CodeRunnerProtocolViolation {
		message = "script worker violated its protocol"
	}
	return failedResponse(request.AttemptID, failureCode, message), nil
}

func (runtime *windowsRuntime) execute(parent context.Context, request Request) (Response, string, error) {
	attemptContext, cancel := context.WithTimeout(
		parent,
		time.Duration(request.TimeoutMillis)*time.Millisecond+workerStartupAllowance,
	)
	defer cancel()

	appContainerSID, err := appContainerSID()
	if err != nil {
		return Response{}, CodeIsolationUnavailable, fmt.Errorf("prepare AppContainer profile: %w", err)
	}
	defer windows.FreeSid(appContainerSID)
	workerExecutable, workerDirectory, err := runtime.prepareWorkerExecutable(appContainerSID)
	if err != nil {
		return Response{}, CodeIsolationUnavailable, fmt.Errorf("stage AppContainer worker executable: %w", err)
	}

	job, err := createConstrainedJob(runtime.options, request)
	if err != nil {
		return Response{}, CodeIsolationUnavailable, fmt.Errorf("create constrained Job Object: %w", err)
	}
	defer windows.CloseHandle(job)

	stdinRead, stdinWrite, err := inheritablePipe()
	if err != nil {
		return Response{}, CodeIsolationUnavailable, fmt.Errorf("create script stdin pipe: %w", err)
	}
	defer closeHandle(&stdinRead)
	defer closeHandle(&stdinWrite)
	stdoutRead, stdoutWrite, err := inheritablePipe()
	if err != nil {
		return Response{}, CodeIsolationUnavailable, fmt.Errorf("create script stdout pipe: %w", err)
	}
	defer closeHandle(&stdoutRead)
	defer closeHandle(&stdoutWrite)
	stderrRead, stderrWrite, err := inheritablePipe()
	if err != nil {
		return Response{}, CodeIsolationUnavailable, fmt.Errorf("create script stderr pipe: %w", err)
	}
	defer closeHandle(&stderrRead)
	defer closeHandle(&stderrWrite)

	if err := windows.SetHandleInformation(stdinWrite, windows.HANDLE_FLAG_INHERIT, 0); err != nil {
		return Response{}, CodeIsolationUnavailable, fmt.Errorf("seal parent script stdin handle: %w", err)
	}
	if err := windows.SetHandleInformation(stdoutRead, windows.HANDLE_FLAG_INHERIT, 0); err != nil {
		return Response{}, CodeIsolationUnavailable, fmt.Errorf("seal parent script stdout handle: %w", err)
	}
	if err := windows.SetHandleInformation(stderrRead, windows.HANDLE_FLAG_INHERIT, 0); err != nil {
		return Response{}, CodeIsolationUnavailable, fmt.Errorf("seal parent script stderr handle: %w", err)
	}

	attributes, err := windows.NewProcThreadAttributeList(5)
	if err != nil {
		return Response{}, CodeIsolationUnavailable, fmt.Errorf("allocate process attribute list: %w", err)
	}
	defer attributes.Delete()

	capabilities := securityCapabilities{AppContainerSID: appContainerSID}
	jobs := []windows.Handle{job}
	childHandles := []windows.Handle{stdinRead, stdoutWrite, stderrWrite}
	allPackagesPolicy := processCreationAllPackagesOptOut
	childProcessPolicy := processCreationChildProcessRestricted
	if err := attributes.Update(
		procThreadAttributeSecurityCapabilities,
		unsafe.Pointer(&capabilities),
		unsafe.Sizeof(capabilities),
	); err != nil {
		return Response{}, CodeIsolationUnavailable, fmt.Errorf("install AppContainer process attribute: %w", err)
	}
	if err := attributes.Update(
		procThreadAttributeJobList,
		unsafe.Pointer(&jobs[0]),
		unsafe.Sizeof(jobs[0]),
	); err != nil {
		return Response{}, CodeIsolationUnavailable, fmt.Errorf("install atomic Job List process attribute: %w", err)
	}
	if err := attributes.Update(
		windows.PROC_THREAD_ATTRIBUTE_HANDLE_LIST,
		unsafe.Pointer(&childHandles[0]),
		uintptr(len(childHandles))*unsafe.Sizeof(childHandles[0]),
	); err != nil {
		return Response{}, CodeIsolationUnavailable, fmt.Errorf("install inherited Handle List process attribute: %w", err)
	}
	if err := attributes.Update(
		procThreadAttributeAllApplicationPackages,
		unsafe.Pointer(&allPackagesPolicy),
		unsafe.Sizeof(allPackagesPolicy),
	); err != nil {
		return Response{}, CodeIsolationUnavailable, fmt.Errorf("install LPAC process attribute: %w", err)
	}
	if err := attributes.Update(
		procThreadAttributeChildProcessPolicy,
		unsafe.Pointer(&childProcessPolicy),
		unsafe.Sizeof(childProcessPolicy),
	); err != nil {
		return Response{}, CodeIsolationUnavailable, fmt.Errorf("install child-process policy attribute: %w", err)
	}

	executable, err := windows.UTF16PtrFromString(workerExecutable)
	if err != nil {
		return Response{}, CodeIsolationUnavailable, fmt.Errorf("encode script worker executable path: %w", err)
	}
	commandLine, err := windows.UTF16FromString(windows.ComposeCommandLine([]string{workerExecutable, WorkerArgument}))
	if err != nil {
		return Response{}, CodeIsolationUnavailable, fmt.Errorf("encode script worker command line: %w", err)
	}
	environment, err := workerEnvironment(workerDirectory)
	if err != nil {
		return Response{}, CodeIsolationUnavailable, fmt.Errorf("build script worker environment: %w", err)
	}
	currentDirectory, err := windows.UTF16PtrFromString(workerDirectory)
	if err != nil {
		return Response{}, CodeIsolationUnavailable, fmt.Errorf("encode script worker directory: %w", err)
	}
	startup := windows.StartupInfoEx{
		StartupInfo: windows.StartupInfo{
			Cb:         uint32(unsafe.Sizeof(windows.StartupInfoEx{})),
			Flags:      windows.STARTF_USESTDHANDLES | windows.STARTF_USESHOWWINDOW,
			ShowWindow: windows.SW_HIDE,
			StdInput:   stdinRead,
			StdOutput:  stdoutWrite,
			StdErr:     stderrWrite,
		},
		ProcThreadAttributeList: attributes.List(),
	}
	var process windows.ProcessInformation
	err = windows.CreateProcess(
		executable,
		&commandLine[0],
		nil,
		nil,
		true,
		windows.CREATE_NO_WINDOW|windows.CREATE_UNICODE_ENVIRONMENT|windows.EXTENDED_STARTUPINFO_PRESENT,
		&environment[0],
		currentDirectory,
		&startup.StartupInfo,
		&process,
	)
	goruntime.KeepAlive(capabilities)
	goruntime.KeepAlive(jobs)
	goruntime.KeepAlive(childHandles)
	goruntime.KeepAlive(allPackagesPolicy)
	goruntime.KeepAlive(childProcessPolicy)
	goruntime.KeepAlive(environment)
	if err != nil {
		return Response{}, CodeIsolationUnavailable, fmt.Errorf("create LPAC script worker process: %w", err)
	}
	defer windows.CloseHandle(process.Process)
	_ = windows.CloseHandle(process.Thread)
	if err := verifyLaunchedProcessConfinement(process.Process); err != nil {
		_ = windows.TerminateJobObject(job, workerTerminationCode)
		return Response{}, CodeIsolationUnavailable, fmt.Errorf("verify launched script worker confinement: %w", err)
	}

	closeHandle(&stdinRead)
	closeHandle(&stdoutWrite)
	closeHandle(&stderrWrite)

	stdinFile := os.NewFile(uintptr(stdinWrite), "script-worker-stdin")
	stdinWrite = 0
	stdoutFile := os.NewFile(uintptr(stdoutRead), "script-worker-stdout")
	stdoutRead = 0
	stderrFile := os.NewFile(uintptr(stderrRead), "script-worker-stderr")
	stderrRead = 0
	defer stdinFile.Close()
	defer stdoutFile.Close()
	defer stderrFile.Close()

	writeDone := make(chan error, 1)
	readDone := make(chan responseResult, 1)
	processDone := make(chan processResult, 1)
	stderrDone := make(chan struct{}, 1)
	go func() {
		err := WriteRequest(stdinFile, request)
		closeErr := stdinFile.Close()
		if err == nil {
			err = closeErr
		}
		writeDone <- err
	}()
	go func() {
		response, err := ReadResponse(stdoutFile)
		readDone <- responseResult{response: response, err: err}
	}()
	go func() {
		_, _ = io.Copy(io.Discard, stderrFile)
		stderrDone <- struct{}{}
	}()
	go func() {
		result := processResult{}
		if event, waitErr := windows.WaitForSingleObject(process.Process, windows.INFINITE); waitErr != nil {
			result.err = waitErr
		} else if event != windows.WAIT_OBJECT_0 {
			result.err = fmt.Errorf("unexpected script worker wait result %d", event)
		} else {
			result.err = windows.GetExitCodeProcess(process.Process, &result.exitCode)
		}
		processDone <- result
	}()

	var (
		writeErr                     error
		readResult                   responseResult
		waitResult                   processResult
		cancelErr                    error
		wrote, read, waited, drained bool
		attemptDone                  = attemptContext.Done()
	)
	for !(wrote && read && waited && drained) {
		select {
		case <-attemptDone:
			_ = windows.TerminateJobObject(job, workerTerminationCode)
			_ = stdinFile.Close()
			_ = stdoutFile.Close()
			_ = stderrFile.Close()
			cancelErr = attemptContext.Err()
			attemptDone = nil
		case writeErr = <-writeDone:
			wrote = true
			if writeErr != nil {
				_ = windows.TerminateJobObject(job, workerTerminationCode)
			}
		case readResult = <-readDone:
			read = true
		case waitResult = <-processDone:
			waited = true
		case <-stderrDone:
			drained = true
		}
	}
	if cancelErr != nil {
		return Response{}, CodeRunnerCrashed, cancelErr
	}
	if writeErr != nil || waitResult.err != nil || waitResult.exitCode != WorkerExitOK {
		return Response{}, CodeRunnerCrashed, errors.Join(writeErr, waitResult.err, exitCodeError(waitResult.exitCode))
	}
	if readResult.err != nil {
		return Response{}, CodeRunnerProtocolViolation, readResult.err
	}
	if readResult.response.AttemptID != request.AttemptID {
		return Response{}, CodeRunnerProtocolViolation, errors.New("script worker attempt identity mismatch")
	}
	return readResult.response, "", nil
}

func verifyLaunchedProcessConfinement(process windows.Handle) error {
	var token windows.Token
	if err := windows.OpenProcessToken(process, windows.TOKEN_QUERY, &token); err != nil {
		return err
	}
	defer token.Close()
	isAppContainer, err := tokenBoolean(token, tokenIsAppContainer)
	if err != nil {
		return fmt.Errorf("query AppContainer token: %w", err)
	}
	if !isAppContainer {
		return errWorkerNotAppContainer
	}
	isLPAC, err := tokenBoolean(token, tokenIsLessPrivilegedAppContainer)
	if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
		isLPAC, err = tokenHasNoAllApplicationPackagesAttribute(token)
	}
	if err != nil {
		return fmt.Errorf("query LPAC token: %w", err)
	}
	if !isLPAC {
		return errWorkerNotLPAC
	}
	return nil
}

func workerEnvironment(profileDirectory string) ([]uint16, error) {
	windowsDirectory, err := windows.GetWindowsDirectory()
	if err != nil {
		return nil, err
	}
	entries := []string{
		"LOCALAPPDATA=" + profileDirectory,
		"SystemRoot=" + windowsDirectory,
		"TEMP=" + profileDirectory,
		"TMP=" + profileDirectory,
		"USERPROFILE=" + profileDirectory,
		"WINDIR=" + windowsDirectory,
	}
	block := make([]uint16, 0, 256)
	for _, entry := range entries {
		encoded, err := windows.UTF16FromString(entry)
		if err != nil {
			return nil, err
		}
		block = append(block, encoded...)
	}
	block = append(block, 0)
	return block, nil
}

func (runtime *windowsRuntime) prepareWorkerExecutable(appContainerSID *windows.SID) (string, string, error) {
	folder, err := appContainerFolder(appContainerSID)
	if err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(folder, 0o700); err != nil {
		return "", "", err
	}
	sourceDigest, sourceSize, err := digestExecutable(runtime.options.Executable)
	if err != nil {
		return "", "", err
	}
	target := filepath.Join(folder, "yotta-script-worker-"+sourceDigest+".exe")
	if targetDigest, targetSize, targetErr := digestExecutable(target); targetErr == nil &&
		targetDigest == sourceDigest && targetSize == sourceSize {
		return target, folder, nil
	}
	_ = os.Remove(target)

	source, err := os.Open(runtime.options.Executable)
	if err != nil {
		return "", "", err
	}
	defer source.Close()
	temporary, err := os.CreateTemp(folder, ".yotta-script-worker-*.tmp")
	if err != nil {
		return "", "", err
	}
	temporaryPath := temporary.Name()
	committed := false
	defer func() {
		_ = temporary.Close()
		if !committed {
			_ = os.Remove(temporaryPath)
		}
	}()
	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(temporary, hash), io.LimitReader(source, maxWorkerExecutableBytes+1))
	if err != nil {
		return "", "", err
	}
	if written != sourceSize || written > maxWorkerExecutableBytes || hex.EncodeToString(hash.Sum(nil)) != sourceDigest {
		return "", "", errors.New("script worker executable changed while staging")
	}
	if err := temporary.Sync(); err != nil {
		return "", "", err
	}
	if err := temporary.Close(); err != nil {
		return "", "", err
	}
	if err := os.Rename(temporaryPath, target); err != nil {
		return "", "", err
	}
	committed = true
	if targetDigest, targetSize, err := digestExecutable(target); err != nil || targetDigest != sourceDigest || targetSize != sourceSize {
		_ = os.Remove(target)
		return "", "", errors.New("staged script worker executable failed integrity verification")
	}
	return target, folder, nil
}

func digestExecutable(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", 0, err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxWorkerExecutableBytes {
		return "", 0, errors.New("script worker executable is not a bounded regular file")
	}
	hash := sha256.New()
	written, err := io.Copy(hash, io.LimitReader(file, maxWorkerExecutableBytes+1))
	if err != nil {
		return "", 0, err
	}
	if written != info.Size() {
		return "", 0, errors.New("script worker executable size changed while hashing")
	}
	return hex.EncodeToString(hash.Sum(nil)), written, nil
}

func appContainerFolder(sid *windows.SID) (string, error) {
	sidString := sid.String()
	if sidString == "" {
		return "", errors.New("format AppContainer SID")
	}
	sidPointer, err := windows.UTF16PtrFromString(sidString)
	if err != nil {
		return "", err
	}
	var path *uint16
	result, _, _ := getAppContainerFolderPath.Call(
		uintptr(unsafe.Pointer(sidPointer)),
		uintptr(unsafe.Pointer(&path)),
	)
	if err := checkHRESULT("get AppContainer folder", result); err != nil {
		return "", err
	}
	if path == nil {
		return "", errors.New("get AppContainer folder returned no path")
	}
	defer windows.CoTaskMemFree(unsafe.Pointer(path))
	folder := filepath.Clean(windows.UTF16PtrToString(path))
	if !filepath.IsAbs(folder) {
		return "", errors.New("AppContainer folder is not absolute")
	}
	return folder, nil
}

func inheritablePipe() (windows.Handle, windows.Handle, error) {
	attributes := windows.SecurityAttributes{
		Length:        uint32(unsafe.Sizeof(windows.SecurityAttributes{})),
		InheritHandle: 1,
	}
	var read, write windows.Handle
	if err := windows.CreatePipe(&read, &write, &attributes, 0); err != nil {
		return 0, 0, err
	}
	return read, write, nil
}

func createConstrainedJob(options RuntimeOptions, request Request) (windows.Handle, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, err
	}
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_ACTIVE_PROCESS |
		windows.JOB_OBJECT_LIMIT_DIE_ON_UNHANDLED_EXCEPTION |
		windows.JOB_OBJECT_LIMIT_JOB_MEMORY |
		windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE |
		windows.JOB_OBJECT_LIMIT_PROCESS_MEMORY |
		windows.JOB_OBJECT_LIMIT_PROCESS_TIME
	limits.BasicLimitInformation.ActiveProcessLimit = 1
	limits.BasicLimitInformation.PerProcessUserTimeLimit = int64(
		(time.Duration(request.TimeoutMillis)*time.Millisecond + workerStartupAllowance) / (100 * time.Nanosecond),
	)
	limits.ProcessMemoryLimit = uintptr(options.ProcessMemoryBytes)
	limits.JobMemoryLimit = uintptr(options.JobMemoryBytes)
	if _, err := windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&limits)),
		uint32(unsafe.Sizeof(limits)),
	); err != nil {
		windows.CloseHandle(job)
		return 0, err
	}
	uiRestrictions := windows.JOBOBJECT_BASIC_UI_RESTRICTIONS{
		UIRestrictionsClass: windows.JOB_OBJECT_UILIMIT_DESKTOP |
			windows.JOB_OBJECT_UILIMIT_DISPLAYSETTINGS |
			windows.JOB_OBJECT_UILIMIT_EXITWINDOWS |
			windows.JOB_OBJECT_UILIMIT_GLOBALATOMS |
			windows.JOB_OBJECT_UILIMIT_HANDLES |
			windows.JOB_OBJECT_UILIMIT_READCLIPBOARD |
			windows.JOB_OBJECT_UILIMIT_SYSTEMPARAMETERS |
			windows.JOB_OBJECT_UILIMIT_WRITECLIPBOARD,
	}
	if _, err := windows.SetInformationJobObject(
		job,
		windows.JobObjectBasicUIRestrictions,
		uintptr(unsafe.Pointer(&uiRestrictions)),
		uint32(unsafe.Sizeof(uiRestrictions)),
	); err != nil {
		windows.CloseHandle(job)
		return 0, err
	}
	return job, nil
}

func appContainerSID() (*windows.SID, error) {
	name, err := windows.UTF16PtrFromString(appContainerProfileName)
	if err != nil {
		return nil, err
	}
	displayName, _ := windows.UTF16PtrFromString("Yotta Script Worker")
	description, _ := windows.UTF16PtrFromString("Isolated zero-authority JavaScript worker")
	var sid *windows.SID
	result, _, _ := createAppContainerProfile.Call(
		uintptr(unsafe.Pointer(name)),
		uintptr(unsafe.Pointer(displayName)),
		uintptr(unsafe.Pointer(description)),
		0,
		0,
		uintptr(unsafe.Pointer(&sid)),
	)
	if uint32(result) == hresultAlreadyExists {
		return callDeriveAppContainerSID(name)
	}
	if err := checkHRESULT("create AppContainer profile", result); err != nil {
		return nil, err
	}
	if sid == nil {
		return nil, errors.New("create AppContainer profile returned no SID")
	}
	return sid, nil
}

func callDeriveAppContainerSID(name *uint16) (*windows.SID, error) {
	var sid *windows.SID
	result, _, _ := deriveAppContainerSID.Call(
		uintptr(unsafe.Pointer(name)),
		uintptr(unsafe.Pointer(&sid)),
	)
	if err := checkHRESULT("derive AppContainer SID", result); err != nil {
		return nil, err
	}
	if sid == nil {
		return nil, errors.New("derive AppContainer SID returned no SID")
	}
	return sid, nil
}

func checkHRESULT(action string, result uintptr) error {
	if uint32(result) == 0 {
		return nil
	}
	return fmt.Errorf("%s: HRESULT 0x%08x", action, uint32(result))
}

func closeHandle(handle *windows.Handle) {
	if handle == nil || *handle == 0 || *handle == windows.InvalidHandle {
		return
	}
	_ = windows.CloseHandle(*handle)
	*handle = 0
}

func exitCodeError(code uint32) error {
	if code == WorkerExitOK {
		return nil
	}
	return fmt.Errorf("script worker exited with code %d", code)
}
