// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build linux freebsd openbsd netbsd

// Package client provides an interface to the AMD SEV-SNP guest device commands.
package client

import (
	"fmt"

	labi "github.com/google/go-sev-guest/client/linuxabi"
	"golang.org/x/sys/unix"
)

// LinuxDevice implements the Device interface with Linux ioctls.
type LinuxDevice struct {
	fd int
}

// Open opens the SEV-SNP guest device from a given path
func (d *LinuxDevice) Open(path string) error {
	fd, err := unix.Open(path, unix.O_RDWR, 0)
	if err != nil {
		d.fd = -1
		return fmt.Errorf("could not open AMD SEV guest device at %s: %v", path, err)
	}
	d.fd = fd
	return nil
}

// OpenDevice opens the SEV-SNP guest device.
func OpenDevice() (*LinuxDevice, error) {
	result := &LinuxDevice{}
	if err := result.Open("/dev/sev-guest"); err != nil {
		return nil, err
	}
	return result, nil
}

// Close closes the SEV-SNP guest device.
func (d *LinuxDevice) Close() error {
	if d.fd == -1 { // Not open
		return nil
	}
	if err := unix.Close(d.fd); err != nil {
		return err
	}
	// Prevent double-close.
	d.fd = -1
	return nil
}

// linuxIoctl performs the ioctl Linux syscall with the sev-guest Linux ABI unsafe pointer
// manipulation contained all in this call.
func linuxIoctl(fd int, command uintptr, req *labi.SnpUserGuestRequest) (uintptr, error) {
        abi := req.ABI()
        result, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), command, uintptr(abi.Pointer()))
        abi.Finish(req)

        // TODO(Issue #5): remove the work around for the kernel bug that writes
        // uninitialized memory back on non-EIO.
        if errno != unix.EIO {
                req.FwErr = 0
        }
        if errno != 0 {
                return 0, errno
        }
        return result, nil
}

// Ioctl sends a command with its wrapped request and response values to the Linux device.
func (d *LinuxDevice) Ioctl(command uintptr, req interface{}) (uintptr, error) {
	switch sreq := req.(type) {
	case *labi.SnpUserGuestRequest:
		return linuxIoctl(d.fd, command, sreq)
	}
	return 0, fmt.Errorf("unexpected request value: %v", req)
}
