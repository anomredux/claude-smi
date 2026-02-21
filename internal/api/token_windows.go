//go:build windows

package api

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	advapi32           = syscall.NewLazyDLL("advapi32.dll")
	procCredEnumerateW = advapi32.NewProc("CredEnumerateW")
	procCredFree       = advapi32.NewProc("CredFree")
)

// sysCredential mirrors Windows CREDENTIAL struct layout.
// https://learn.microsoft.com/en-us/windows/win32/api/wincred/ns-wincred-credentialw
type sysCredential struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWritten        [8]byte // FILETIME
	CredentialBlobSize uint32
	CredentialBlob     uintptr
	Persist            uint32
	AttributeCount     uint32
	Attributes         uintptr
	TargetAlias        *uint16
	UserName           *uint16
}

// getOAuthToken reads the Claude Code OAuth token from Windows Credential Manager.
func getOAuthToken() (string, error) {
	filter, err := syscall.UTF16PtrFromString(keychainLabel + "*")
	if err != nil {
		return "", fmt.Errorf("invalid filter: %w", err)
	}

	var count uint32
	var creds uintptr
	ret, _, _ := procCredEnumerateW.Call(
		uintptr(unsafe.Pointer(filter)),
		0,
		uintptr(unsafe.Pointer(&count)),
		uintptr(unsafe.Pointer(&creds)),
	)
	if ret == 0 || count == 0 {
		return "", fmt.Errorf("no credentials found for %q in Windows Credential Manager", keychainLabel)
	}
	defer procCredFree.Call(creds)

	// Try each matching credential until we find a valid token.
	ptrs := (*[1 << 10]*sysCredential)(unsafe.Pointer(creds))
	for i := uint32(0); i < count; i++ {
		cred := ptrs[i]
		if cred.CredentialBlobSize == 0 {
			continue
		}
		blob := (*[1 << 20]byte)(unsafe.Pointer(cred.CredentialBlob))[:cred.CredentialBlobSize:cred.CredentialBlobSize]
		token, err := parseCredentialJSON(string(blob))
		if err == nil {
			return token, nil
		}
	}
	return "", fmt.Errorf("no valid OAuth token found in Windows Credential Manager")
}
