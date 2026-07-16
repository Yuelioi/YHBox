//go:build windows

package processsandbox

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
	"strings"
	"sync"
	"time"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	terminationCode = 0x59303131

	procThreadAttributeSecurityCapabilities          = 0x00020009
	procThreadAttributeJobList                       = 0x0002000d
	procThreadAttributeChildProcessPolicy            = 0x0002000e
	procThreadAttributeAllApplicationPackages        = 0x0002000f
	processCreationChildProcessRestricted     uint32 = 0x00000001
	processCreationAllPackagesOptOut          uint32 = 0x00000001
	hresultAlreadyExists                             = 0x800700b7

	tokenIsAppContainer               = 29
	tokenSecurityAttributes           = 39
	tokenIsLessPrivilegedAppContainer = 46
	tokenSecurityAttributeUint64      = 0x0002
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

type windowsRunner struct {
	options Options
	mu      sync.Mutex
}

func newPlatformRunner(options Options) platformRunner { return &windowsRunner{options: options} }

func (runner *windowsRunner) start(ctx context.Context, request Request) (platformProcess, error) {
	runner.mu.Lock()
	defer runner.mu.Unlock()

	sid, err := appContainerSID(runner.options)
	if err != nil {
		return nil, fmt.Errorf("%w: prepare AppContainer profile: %v", ErrIsolationUnavailable, err)
	}
	defer windows.FreeSid(sid)
	executable, directory, err := stageImage(sid, request.Image)
	if err != nil {
		return nil, fmt.Errorf("%w: stage AppContainer image: %v", ErrIsolationUnavailable, err)
	}
	job, err := createConstrainedJob(runner.options, request.Timeout)
	if err != nil {
		return nil, fmt.Errorf("%w: create constrained Job Object: %v", ErrIsolationUnavailable, err)
	}
	launched := false
	defer func() {
		if !launched {
			_ = windows.CloseHandle(job)
		}
	}()

	stdinRead, stdinWrite, err := inheritablePipe()
	if err != nil {
		return nil, fmt.Errorf("%w: create stdin pipe: %v", ErrIsolationUnavailable, err)
	}
	defer closeHandle(&stdinRead)
	defer closeHandle(&stdinWrite)
	stdoutRead, stdoutWrite, err := inheritablePipe()
	if err != nil {
		return nil, fmt.Errorf("%w: create stdout pipe: %v", ErrIsolationUnavailable, err)
	}
	defer closeHandle(&stdoutRead)
	defer closeHandle(&stdoutWrite)
	stderrRead, stderrWrite, err := inheritablePipe()
	if err != nil {
		return nil, fmt.Errorf("%w: create stderr pipe: %v", ErrIsolationUnavailable, err)
	}
	defer closeHandle(&stderrRead)
	defer closeHandle(&stderrWrite)
	for _, handle := range []windows.Handle{stdinWrite, stdoutRead, stderrRead} {
		if err := windows.SetHandleInformation(handle, windows.HANDLE_FLAG_INHERIT, 0); err != nil {
			return nil, fmt.Errorf("%w: seal parent pipe handle: %v", ErrIsolationUnavailable, err)
		}
	}

	attributes, err := windows.NewProcThreadAttributeList(5)
	if err != nil {
		return nil, fmt.Errorf("%w: allocate process attribute list: %v", ErrIsolationUnavailable, err)
	}
	defer attributes.Delete()
	capabilities := securityCapabilities{AppContainerSID: sid}
	jobs := []windows.Handle{job}
	childHandles := []windows.Handle{stdinRead, stdoutWrite, stderrWrite}
	allPackagesPolicy := processCreationAllPackagesOptOut
	childProcessPolicy := processCreationChildProcessRestricted
	updates := []struct {
		attribute uintptr
		value     unsafe.Pointer
		size      uintptr
		label     string
	}{
		{procThreadAttributeSecurityCapabilities, unsafe.Pointer(&capabilities), unsafe.Sizeof(capabilities), "AppContainer"},
		{procThreadAttributeJobList, unsafe.Pointer(&jobs[0]), unsafe.Sizeof(jobs[0]), "atomic Job List"},
		{windows.PROC_THREAD_ATTRIBUTE_HANDLE_LIST, unsafe.Pointer(&childHandles[0]), uintptr(len(childHandles)) * unsafe.Sizeof(childHandles[0]), "inherited Handle List"},
		{procThreadAttributeAllApplicationPackages, unsafe.Pointer(&allPackagesPolicy), unsafe.Sizeof(allPackagesPolicy), "LPAC"},
		{procThreadAttributeChildProcessPolicy, unsafe.Pointer(&childProcessPolicy), unsafe.Sizeof(childProcessPolicy), "child-process policy"},
	}
	for _, update := range updates {
		if err := attributes.Update(update.attribute, update.value, update.size); err != nil {
			return nil, fmt.Errorf("%w: install %s process attribute: %v", ErrIsolationUnavailable, update.label, err)
		}
	}

	executablePointer, err := windows.UTF16PtrFromString(executable)
	if err != nil {
		return nil, fmt.Errorf("%w: encode image path: %v", ErrIsolationUnavailable, err)
	}
	arguments := append([]string{executable}, request.Args...)
	commandLine, err := windows.UTF16FromString(windows.ComposeCommandLine(arguments))
	if err != nil {
		return nil, fmt.Errorf("%w: encode command line: %v", ErrIsolationUnavailable, err)
	}
	environment, err := sandboxEnvironment(directory)
	if err != nil {
		return nil, fmt.Errorf("%w: build environment: %v", ErrIsolationUnavailable, err)
	}
	currentDirectory, err := windows.UTF16PtrFromString(directory)
	if err != nil {
		return nil, fmt.Errorf("%w: encode working directory: %v", ErrIsolationUnavailable, err)
	}
	startup := windows.StartupInfoEx{
		StartupInfo: windows.StartupInfo{
			Cb: uint32(unsafe.Sizeof(windows.StartupInfoEx{})), Flags: windows.STARTF_USESTDHANDLES | windows.STARTF_USESHOWWINDOW,
			ShowWindow: windows.SW_HIDE, StdInput: stdinRead, StdOutput: stdoutWrite, StdErr: stderrWrite,
		},
		ProcThreadAttributeList: attributes.List(),
	}
	var information windows.ProcessInformation
	err = windows.CreateProcess(
		executablePointer, &commandLine[0], nil, nil, true,
		windows.CREATE_NO_WINDOW|windows.CREATE_UNICODE_ENVIRONMENT|windows.EXTENDED_STARTUPINFO_PRESENT,
		&environment[0], currentDirectory, &startup.StartupInfo, &information,
	)
	goruntime.KeepAlive(capabilities)
	goruntime.KeepAlive(jobs)
	goruntime.KeepAlive(childHandles)
	goruntime.KeepAlive(allPackagesPolicy)
	goruntime.KeepAlive(childProcessPolicy)
	goruntime.KeepAlive(environment)
	if err != nil {
		return nil, fmt.Errorf("%w: create LPAC process: %v", ErrIsolationUnavailable, err)
	}
	_ = windows.CloseHandle(information.Thread)
	if err := verifyProcessConfinement(information.Process); err != nil {
		_ = windows.TerminateJobObject(job, terminationCode)
		_ = windows.CloseHandle(information.Process)
		return nil, fmt.Errorf("%w: verify launched process confinement: %v", ErrIsolationUnavailable, err)
	}

	closeHandle(&stdinRead)
	closeHandle(&stdoutWrite)
	closeHandle(&stderrWrite)
	process := &windowsProcess{
		stdinFile: os.NewFile(uintptr(stdinWrite), "sandbox-stdin"), stdoutFile: os.NewFile(uintptr(stdoutRead), "sandbox-stdout"),
		stderrFile: os.NewFile(uintptr(stderrRead), "sandbox-stderr"), process: information.Process, job: job, done: make(chan struct{}),
	}
	stdinWrite, stdoutRead, stderrRead = 0, 0, 0
	launched = true
	go process.await()
	if err := ctx.Err(); err != nil {
		_ = process.terminate()
		_ = process.close()
		return nil, err
	}
	return process, nil
}

type windowsProcess struct {
	stdinFile  *os.File
	stdoutFile *os.File
	stderrFile *os.File
	process    windows.Handle
	job        windows.Handle
	done       chan struct{}
	result     Result
	waitErr    error
	stopOnce   sync.Once
	closeOnce  sync.Once
}

func (process *windowsProcess) stdin() io.WriteCloser { return process.stdinFile }
func (process *windowsProcess) stdout() io.ReadCloser { return process.stdoutFile }
func (process *windowsProcess) stderr() io.ReadCloser { return process.stderrFile }

func (process *windowsProcess) await() {
	event, err := windows.WaitForSingleObject(process.process, windows.INFINITE)
	if err != nil {
		process.waitErr = err
	} else if event != windows.WAIT_OBJECT_0 {
		process.waitErr = fmt.Errorf("unexpected sandbox wait result %d", event)
	} else {
		process.waitErr = windows.GetExitCodeProcess(process.process, &process.result.ExitCode)
	}
	close(process.done)
}

func (process *windowsProcess) wait() (Result, error) {
	<-process.done
	return process.result, process.waitErr
}

func (process *windowsProcess) terminate() error {
	var err error
	process.stopOnce.Do(func() { err = windows.TerminateJobObject(process.job, terminationCode) })
	return err
}

func (process *windowsProcess) close() error {
	var result error
	process.closeOnce.Do(func() {
		_ = process.terminate()
		result = errors.Join(process.stdinFile.Close(), process.stdoutFile.Close(), process.stderrFile.Close())
		<-process.done
		result = errors.Join(result, windows.CloseHandle(process.process), windows.CloseHandle(process.job))
	})
	return result
}

func stageImage(sid *windows.SID, image Image) (string, string, error) {
	folder, err := appContainerFolder(sid)
	if err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(folder, 0o700); err != nil {
		return "", "", err
	}
	base := strings.TrimSuffix(image.name, filepath.Ext(image.name))
	target := filepath.Join(folder, base+"-"+image.digest+".exe")
	if matchesImage(target, image) {
		return target, folder, nil
	}
	_ = os.Remove(target)
	temporary, err := os.CreateTemp(folder, ".yotta-sandbox-*.tmp")
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
	if _, err := temporary.Write(image.content); err != nil {
		return "", "", err
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
	if !matchesImage(target, image) {
		_ = os.Remove(target)
		return "", "", errors.New("staged sandbox image failed integrity verification")
	}
	return target, folder, nil
}

func matchesImage(path string, image Image) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() != int64(len(image.content)) {
		return false
	}
	hash := sha256.New()
	written, err := io.Copy(hash, io.LimitReader(file, MaxImageBytes+1))
	return err == nil && written == info.Size() && hex.EncodeToString(hash.Sum(nil)) == image.digest
}

func appContainerSID(options Options) (*windows.SID, error) {
	name, err := windows.UTF16PtrFromString(options.ProfileName)
	if err != nil {
		return nil, err
	}
	displayName, err := windows.UTF16PtrFromString(options.DisplayName)
	if err != nil {
		return nil, err
	}
	description, err := windows.UTF16PtrFromString(options.Description)
	if err != nil {
		return nil, err
	}
	var sid *windows.SID
	result, _, _ := createAppContainerProfile.Call(
		uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(displayName)), uintptr(unsafe.Pointer(description)),
		0, 0, uintptr(unsafe.Pointer(&sid)),
	)
	if uint32(result) == hresultAlreadyExists {
		return deriveSID(name)
	}
	if err := checkHRESULT("create AppContainer profile", result); err != nil {
		return nil, err
	}
	if sid == nil {
		return nil, errors.New("create AppContainer profile returned no SID")
	}
	return sid, nil
}

func deriveSID(name *uint16) (*windows.SID, error) {
	var sid *windows.SID
	result, _, _ := deriveAppContainerSID.Call(uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(&sid)))
	if err := checkHRESULT("derive AppContainer SID", result); err != nil {
		return nil, err
	}
	if sid == nil {
		return nil, errors.New("derive AppContainer SID returned no SID")
	}
	return sid, nil
}

func appContainerFolder(sid *windows.SID) (string, error) {
	pointer, err := windows.UTF16PtrFromString(sid.String())
	if err != nil {
		return "", err
	}
	var path *uint16
	result, _, _ := getAppContainerFolderPath.Call(uintptr(unsafe.Pointer(pointer)), uintptr(unsafe.Pointer(&path)))
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

func sandboxEnvironment(profileDirectory string) ([]uint16, error) {
	windowsDirectory, err := windows.GetWindowsDirectory()
	if err != nil {
		return nil, err
	}
	entries := []string{
		"LOCALAPPDATA=" + profileDirectory, "SystemRoot=" + windowsDirectory, "TEMP=" + profileDirectory,
		"TMP=" + profileDirectory, "USERPROFILE=" + profileDirectory, "WINDIR=" + windowsDirectory,
	}
	block := make([]uint16, 0, 256)
	for _, entry := range entries {
		encoded, err := windows.UTF16FromString(entry)
		if err != nil {
			return nil, err
		}
		block = append(block, encoded...)
	}
	return append(block, 0), nil
}

func inheritablePipe() (windows.Handle, windows.Handle, error) {
	attributes := windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), InheritHandle: 1}
	var read, write windows.Handle
	if err := windows.CreatePipe(&read, &write, &attributes, 0); err != nil {
		return 0, 0, err
	}
	return read, write, nil
}

func createConstrainedJob(options Options, timeout time.Duration) (windows.Handle, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, err
	}
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_ACTIVE_PROCESS |
		windows.JOB_OBJECT_LIMIT_DIE_ON_UNHANDLED_EXCEPTION | windows.JOB_OBJECT_LIMIT_JOB_MEMORY |
		windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE | windows.JOB_OBJECT_LIMIT_PROCESS_MEMORY |
		windows.JOB_OBJECT_LIMIT_PROCESS_TIME
	limits.BasicLimitInformation.ActiveProcessLimit = 1
	limits.BasicLimitInformation.PerProcessUserTimeLimit = int64(timeout / (100 * time.Nanosecond))
	limits.ProcessMemoryLimit = uintptr(options.ProcessMemoryBytes)
	limits.JobMemoryLimit = uintptr(options.JobMemoryBytes)
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); err != nil {
		_ = windows.CloseHandle(job)
		return 0, err
	}
	ui := windows.JOBOBJECT_BASIC_UI_RESTRICTIONS{UIRestrictionsClass: windows.JOB_OBJECT_UILIMIT_DESKTOP |
		windows.JOB_OBJECT_UILIMIT_DISPLAYSETTINGS | windows.JOB_OBJECT_UILIMIT_EXITWINDOWS |
		windows.JOB_OBJECT_UILIMIT_GLOBALATOMS | windows.JOB_OBJECT_UILIMIT_HANDLES |
		windows.JOB_OBJECT_UILIMIT_READCLIPBOARD | windows.JOB_OBJECT_UILIMIT_SYSTEMPARAMETERS |
		windows.JOB_OBJECT_UILIMIT_WRITECLIPBOARD}
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectBasicUIRestrictions, uintptr(unsafe.Pointer(&ui)), uint32(unsafe.Sizeof(ui))); err != nil {
		_ = windows.CloseHandle(job)
		return 0, err
	}
	return job, nil
}

func verifyProcessConfinement(process windows.Handle) error {
	var token windows.Token
	if err := windows.OpenProcessToken(process, windows.TOKEN_QUERY, &token); err != nil {
		return err
	}
	defer token.Close()
	isAppContainer, err := tokenBoolean(token, tokenIsAppContainer)
	if err != nil {
		return err
	}
	if !isAppContainer {
		return ErrNotAppContainer
	}
	isLPAC, err := tokenBoolean(token, tokenIsLessPrivilegedAppContainer)
	if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
		isLPAC, err = tokenHasNoAllApplicationPackagesAttribute(token)
	}
	if err != nil {
		return err
	}
	if !isLPAC {
		return ErrNotLPAC
	}
	return nil
}

type tokenSecurityAttributesInformation struct {
	Version        uint16
	Reserved       uint16
	AttributeCount uint32
	Attributes     unsafe.Pointer
}
type tokenUnicodeString struct {
	Length        uint16
	MaximumLength uint16
	Buffer        *uint16
}
type tokenSecurityAttributeV1 struct {
	Name       tokenUnicodeString
	ValueType  uint16
	Reserved   uint16
	Flags      uint32
	ValueCount uint32
	Values     unsafe.Pointer
}

func tokenBoolean(token windows.Token, informationClass uint32) (bool, error) {
	var value, returned uint32
	if err := windows.GetTokenInformation(token, informationClass, (*byte)(unsafe.Pointer(&value)), uint32(unsafe.Sizeof(value)), &returned); err != nil {
		return false, err
	}
	if returned != 0 && returned != uint32(unsafe.Sizeof(value)) {
		return false, errors.New("unexpected sandbox token information size")
	}
	return value != 0, nil
}

func tokenHasNoAllApplicationPackagesAttribute(token windows.Token) (bool, error) {
	var size uint32
	err := windows.GetTokenInformation(token, tokenSecurityAttributes, nil, 0, &size)
	if !errors.Is(err, windows.ERROR_INSUFFICIENT_BUFFER) || size == 0 {
		return false, err
	}
	buffer := make([]byte, size)
	if err := windows.GetTokenInformation(token, tokenSecurityAttributes, &buffer[0], size, &size); err != nil {
		return false, err
	}
	start, end := uintptr(unsafe.Pointer(&buffer[0])), uintptr(unsafe.Pointer(&buffer[0]))+uintptr(len(buffer))
	information := (*tokenSecurityAttributesInformation)(unsafe.Pointer(&buffer[0]))
	if information.Version != 1 || information.AttributeCount > 128 {
		return false, errors.New("invalid token security attributes header")
	}
	attributeBytes := uintptr(information.AttributeCount) * unsafe.Sizeof(tokenSecurityAttributeV1{})
	if information.AttributeCount != 0 && !pointerWithin(information.Attributes, attributeBytes, start, end) {
		return false, errors.New("token security attributes escape their buffer")
	}
	attributes := unsafe.Slice((*tokenSecurityAttributeV1)(information.Attributes), int(information.AttributeCount))
	for _, attribute := range attributes {
		name, err := boundedUnicodeString(attribute.Name, start, end)
		if err != nil {
			return false, err
		}
		if strings.EqualFold(name, "WIN://NOALLAPPPKG") {
			if attribute.Reserved != 0 || attribute.ValueType != tokenSecurityAttributeUint64 || attribute.ValueCount != 1 ||
				!pointerWithin(attribute.Values, unsafe.Sizeof(uint64(0)), start, end) {
				return false, errors.New("invalid WIN://NOALLAPPPKG token attribute")
			}
			return *(*uint64)(attribute.Values) != 0, nil
		}
	}
	return false, nil
}

func pointerWithin(pointer unsafe.Pointer, size, start, end uintptr) bool {
	address := uintptr(pointer)
	return pointer != nil && address >= start && address <= end && size <= end-address
}

func boundedUnicodeString(value tokenUnicodeString, start, end uintptr) (string, error) {
	if value.Length > value.MaximumLength || value.Length%2 != 0 || value.Length > 512 {
		return "", errors.New("token attribute name has invalid length")
	}
	if value.Length == 0 {
		return "", nil
	}
	if !pointerWithin(unsafe.Pointer(value.Buffer), uintptr(value.Length), start, end) {
		return "", errors.New("token attribute name escapes its buffer")
	}
	return string(utf16.Decode(unsafe.Slice(value.Buffer, int(value.Length/2)))), nil
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
