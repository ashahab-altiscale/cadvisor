// +build linux
//
// Provides Filesystem Stats
package fs

/*
 extern int getBytesFree(const char *path, unsigned long long *bytes);
 extern int getBytesTotal(const char *path, unsigned long long *bytes);
*/
import "C"

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"github.com/docker/docker/pkg/mount"
	"github.com/golang/glog"
)

type partition struct {
	mountpoint string
	major      uint
	minor      uint
}

type RealFsInfo struct {
	partitions map[string]partition
}

func NewFsInfo() (FsInfo, error) {
	mounts, err := mount.GetMounts()
	if err != nil {
		return nil, err
	}
	partitions := make(map[string]partition, 0)
	for _, mount := range mounts {
		if !strings.HasPrefix(mount.Fstype, "ext") {
			continue
		}
		// Avoid bind mounts.
		if _, ok := partitions[mount.Source]; ok {
			continue
		}
		partitions[mount.Source] = partition{mount.Mountpoint, uint(mount.Major), uint(mount.Minor)}
	}
	return &RealFsInfo{partitions}, nil
}

func (self *RealFsInfo) GetFsInfoForPath(mountSet map[string]struct{}) ([]Fs, error) {
	filesystems := make([]Fs, 0)
	deviceSet := make(map[string]struct{})
	glog.Infof("Mountset: %s", mountSet)
	for device, partition := range self.partitions {
		_, hasMount := mountSet[partition.mountpoint]
		_, hasDevice := deviceSet[device]
		glog.Infof("mount %s, hasMount %s, device %s, hasDecice %s", partition.mountpoint, hasMount, device, hasDevice)
		if mountSet == nil ||  hasMount &&  !hasDevice {
			total, free, err := getVfsStats(partition.mountpoint)
			if err != nil {
				glog.Errorf("Statvfs failed. Error: %v", err)
			} else {
				deviceSet[device] = struct{}{}
				deviceInfo := DeviceInfo{
					Device: device,
					Major:  uint(partition.major),
					Minor:  uint(partition.minor),
				}
				fs := Fs{deviceInfo, total, free}
				filesystems = append(filesystems, fs)
			}
		}
	}
	return filesystems, nil
}

func (self *RealFsInfo) GetGlobalFsInfo() ([]Fs, error) {
	return self.GetFsInfoForPath(nil)
}

func major(devNumber uint64) uint {
	return uint((devNumber >> 8) & 0xfff)
}

func minor(devNumber uint64) uint {
	return uint((devNumber & 0xff) | ((devNumber >> 12) & 0xfff00))
}

func (self *RealFsInfo) GetDirFsDevice(dir string) (*DeviceInfo, error) {
	var buf syscall.Stat_t
	err := syscall.Stat(dir, &buf)
	if err != nil {
		return nil, fmt.Errorf("stat failed on %s with error: %s", dir, err)
	}
	major := major(buf.Dev)
	minor := minor(buf.Dev)
	for device, partition := range self.partitions {
		if partition.major == major && partition.minor == minor {
			return &DeviceInfo{device, major, minor}, nil
		}
	}
	return nil, fmt.Errorf("could not find device with major: %d, minor: %d in cached partitions map", major, minor)
}

func (self *RealFsInfo) GetDirUsage(dir string) (uint64, error) {
	out, err := exec.Command("du", "-s", dir).CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("du command failed on %s with output %s - %s", dir, out, err)
	}
	usageInKb, err := strconv.ParseUint(strings.Fields(string(out))[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("cannot parse 'du' output %s - %s", out, err)
	}
	return usageInKb * 1024, nil
}

func getVfsStats(path string) (total uint64, free uint64, err error) {
	_p0, err := syscall.BytePtrFromString(path)
	if err != nil {
		return 0, 0, err
	}
	res, err := C.getBytesFree((*C.char)(unsafe.Pointer(_p0)), (*_Ctype_ulonglong)(unsafe.Pointer(&free)))
	if res != 0 {
		return 0, 0, err
	}
	res, err = C.getBytesTotal((*C.char)(unsafe.Pointer(_p0)), (*_Ctype_ulonglong)(unsafe.Pointer(&total)))
	if res != 0 {
		return 0, 0, err
	}
	return total, free, nil
}
