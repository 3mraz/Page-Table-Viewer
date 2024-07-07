package utils

/*
#cgo CFLAGS: -I../src
#cgo LDFLAGS: -L../src -lPTEdit -Wl,-rpath=../src
#include "ptedit_header.h"
*/
import "C"

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unsafe"
)

func InitEntry(cEntry C.ptedit_entry_t) Entry {
	var entry Entry
	entry.pid = uint64(cEntry.pid)
	entry.vaddr = uintptr(cEntry.vaddr)
	entry.pgd = uint64(binary.LittleEndian.Uint64(cEntry.anon0[:8]))
	entry.p4d = uint64(binary.LittleEndian.Uint64(cEntry.anon1[:8]))
	entry.pud = uint64(binary.LittleEndian.Uint64(cEntry.anon2[:8]))
	entry.pmd = uint64(binary.LittleEndian.Uint64(cEntry.anon3[:8]))
	entry.pte = uint64(cEntry.pte)
	entry.valid = uint64(cEntry.valid)
	entry.cEntry = cEntry
	return entry
}

type Entry struct {
	pid    uint64
	vaddr  uintptr
	pgd    uint64
	p4d    uint64
	pud    uint64
	pmd    uint64
	pte    uint64
	valid  uint64
	cEntry C.ptedit_entry_t
}

type PTEntry struct {
	Color string
	P     bool   // present
	W     bool   // writable
	U     bool   // userspace addressable
	Wt    bool   // write through
	Dc    bool   // disabled cache
	A     bool   // accessed
	D     bool   // dirty
	H     bool   // huge page
	Pat   bool   // PAT (2MB or 4MB)
	G     bool   // global TLB entry
	S1    bool   // software 1
	S2    bool   // software 2
	S3    bool   // software 3
	PatL  bool   // huge page (1GB or 2MB)
	Pfn   string // Page Frame Number
	Vfn   string
	S4    bool // software 4
	Kp0   bool // key protection 0
	Kp1   bool // key protection 1
	Kp2   bool // key protection 2
	Kp3   bool // key protection 3
	Nx    bool // no Execute
}

func Virt2Phys(virtAddr string, pid uint64) string {
	virt, prefixFound := strings.CutPrefix(virtAddr, "0x")
	if !prefixFound {
		panic("Virtual address should start with 0x")
	}
	virtAsInt, err := strconv.ParseUint(virt, 16, 64)
	if err != nil {
		panic(err)
	}
	// fmt.Println(virtAsInt)
	phys := uint64(C.virt_2_phys(unsafe.Pointer(uintptr(virtAsInt)), C.size_t(pid)))
	return fmt.Sprintf("0x%x", phys)
}

func PteditKernelImpl() {
	C.ptedit_use_implementation(C.PTEDIT_IMPL_KERNEL)
}

func PteditInit() int {
	return int(C.ptedit_init())
}

func PteditCleanup() {
	C.ptedit_cleanup()
}

func GetRootPhysAddr(pid uint64) uintptr {
	return uintptr(C.ptedit_get_paging_root(C.int(pid)))
}

func GetSystemPageSize() uint64 {
	return uint64(C.ptedit_get_pagesize())
}

func ParsePTEntry(entry uint64, vaddr uint64) PTEntry {
	var e PTEntry
	if int(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_PRESENT)) == 1 {
		e.P = true
	}
	if int(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_RW)) == 1 {
		e.W = true
	}
	if int(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_USER)) == 1 {
		e.U = true
	}
	if int(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_PWT)) == 1 {
		e.Wt = true
	}
	if int(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_PCD)) == 1 {
		e.Dc = true
	}
	if int(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_ACCESSED)) == 1 {
		e.A = true
	}
	if int(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_DIRTY)) == 1 {
		e.D = true
	}
	if int(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_PSE)) == 1 {
		e.Pat = true
	}
	if int(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_GLOBAL)) == 1 {
		e.G = true
	}
	if int(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_SOFTW1)) == 1 {
		e.S1 = true
	}
	if int(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_SOFTW2)) == 1 {
		e.S2 = true
	}
	if int(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_SOFTW3)) == 1 {
		e.S3 = true
	}
	if int(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_PAT_LARGE)) == 1 {
		e.PatL = true
	}
	if int(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_SOFTW4)) == 1 {
		e.S4 = true
	}
	if int(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_PKEY_BIT0)) == 1 {
		e.Kp0 = true
	}
	if int(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_PKEY_BIT1)) == 1 {
		e.Kp1 = true
	}
	if int(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_PKEY_BIT2)) == 1 {
		e.Kp2 = true
	}
	if int(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_PKEY_BIT3)) == 1 {
		e.Kp3 = true
	}
	if int(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_NX)) == 1 {
		e.Nx = true
	}
	e.Pfn = fmt.Sprintf("0x%x", (entry>>12)&uint64((uint64(1)<<40)-1))
	e.Vfn = fmt.Sprintf("0x%x", (vaddr))
	if e.P {
		e.Color = "green"
	} else {
		e.Color = "red"
	}
	return e
}

func GetFirstLvl(pid uint64) []PTEntry {
	var ptEntries []PTEntry
	entries := C.get_mapped_PML4_entries(C.size_t(pid))
	defer C.free(unsafe.Pointer(entries))
	length := int(C.FIRST_LEVEL_ENTRIES)
	goEntries := (*[1 << 30]C.PTEntry)(unsafe.Pointer(entries))[:length:length]
	for i, v := range goEntries {
		if goEntries[i].entry != 0 {
			ptEntries = append(ptEntries, ParsePTEntry(uint64(v.entry), uint64(v.vaddr)))
		}
	}
	return ptEntries
}

func GetSecondLvl(pid uint64, pml4i int64) []PTEntry {
	var ptEntries []PTEntry
	entries := C.get_mapped_PDPT_entries(C.size_t(pid), C.size_t(pml4i))
	defer C.free(unsafe.Pointer(entries))
	length := 512
	goEntries := (*[1 << 30]C.PTEntry)(unsafe.Pointer(entries))[:length:length]
	for i, v := range goEntries {
		if goEntries[i].entry != 0 {
			ptEntries = append(ptEntries, ParsePTEntry(uint64(v.entry), uint64(v.vaddr)))
		}
	}
	return ptEntries
}

func GetThirdLvl(pid uint64, pml4i int64, pdpti int64) []PTEntry {
	var ptEntries []PTEntry
	entries := C.get_mapped_PD_entries(C.size_t(pid), C.size_t(pml4i), C.size_t(pdpti))
	defer C.free(unsafe.Pointer(entries))
	length := 512
	goEntries := (*[1 << 30]C.PTEntry)(unsafe.Pointer(entries))[:length:length]
	for i, v := range goEntries {
		if goEntries[i].entry != 0 {
			ptEntries = append(ptEntries, ParsePTEntry(uint64(v.entry), uint64(v.vaddr)))
		}
	}
	return ptEntries
}

func GetFourthLvl(pid uint64, pml4i int64, pdpti int64, pdi int64) []PTEntry {
	var ptEntries []PTEntry
	entries := C.get_PTE_entries(C.size_t(pid), C.size_t(pml4i), C.size_t(pdpti), C.size_t(pdi))
	defer C.free(unsafe.Pointer(entries))
	length := 512
	goEntries := (*[1 << 30]C.PTEntry)(unsafe.Pointer(entries))[:length:length]
	for i, v := range goEntries {
		if goEntries[i].entry != 0 {
			ptEntries = append(ptEntries, ParsePTEntry(uint64(v.entry), uint64(v.vaddr)))
		}
	}
	return ptEntries
}

func PrintStruct(ptEntry PTEntry) {
	fmt.Println("---------------------------- PTEntry ----------------------------------")
	t := reflect.TypeOf(ptEntry)
	v := reflect.ValueOf(ptEntry)
	for i := 0; i < t.NumField(); i++ {
		fieldName := t.Field(i).Name
		fieldValue := v.Field(i).Interface()
		fmt.Printf("%s: %v\n", fieldName, fieldValue)
	}
}

func numOfEntriesPerLvl() int {
	return int(C.num_entries_per_lvl())
}
