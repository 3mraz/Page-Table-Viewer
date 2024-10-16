package utils

/*
#cgo CFLAGS: -I../src
#cgo LDFLAGS: -L../src -lPTEdit -Wl,-rpath=../src
#include "ptedit_header.h"
*/
import "C"

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unsafe"
)

// type Entry struct {
// 	pid    uint64
// 	vaddr  uintptr
// 	pgd    uint64
// 	p4d    uint64
// 	pud    uint64
// 	pmd    uint64
// 	pte    uint64
// 	valid  uint64
// 	cEntry C.ptedit_entry_t
// }

// Index of each entry for every level
type Translation struct {
	Pml4 string `json:"pml4"`
	Pdpt string `json:"pdpt"`
	Pd   string `json:"pd"`
	Pte  string `json:"pte"`
}

// For memory dump
type Page struct {
	Content     string      `json:"content"`
	Vfn         string      `json:"vfn"`
	Translation Translation `json:"translation"`
}

type MemPage struct {
	PageBytes []byte
	Nx        string
	Addresses [256]string
	Offset    uint64
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

type CodeSection struct {
	Name   string
	Offset uint64
	Code   string
}

func (entry PTEntry) toggleColor() PTEntry {
	if entry.Color == "green-300" {
		entry.Color = "red-300"
	} else {
		entry.Color = "green-300"
	}
	return entry
}

func ValidVirt(virtAddr string) bool {
	re := regexp.MustCompile("^0x[0-9a-fA-F]{2,12}$")
	return re.MatchString(virtAddr)
}

func ValidPhys(physAddr string) bool {
	re := regexp.MustCompile("^0x[0-9a-fA-F]{1,6}$")
	return re.MatchString(physAddr)
}

func replacePfn(pte, pfn uint64) uint64 {
	// Mask to clear the PFN bits (PFN is in bits 12 through 51)
	const pfnMask uint64 = 0x000FFFFFFFFFF000

	// Clear the PFN bits in the PTE
	pte &= ^pfnMask

	// Insert the new PFN into the PTE (shifted into the correct position)
	pte |= (pfn << 12) & pfnMask

	return pte
}

// func InitEntry(cEntry C.ptedit_entry_t) Entry {
// 	var entry Entry
// 	entry.pid = uint64(cEntry.pid)
// 	entry.vaddr = uintptr(cEntry.vaddr)
// 	entry.pgd = uint64(binary.LittleEndian.Uint64(cEntry.anon0[:8]))
// 	entry.p4d = uint64(binary.LittleEndian.Uint64(cEntry.anon1[:8]))
// 	entry.pud = uint64(binary.LittleEndian.Uint64(cEntry.anon2[:8]))
// 	entry.pmd = uint64(binary.LittleEndian.Uint64(cEntry.anon3[:8]))
// 	entry.pte = uint64(cEntry.pte)
// 	entry.valid = uint64(cEntry.valid)
// 	entry.cEntry = cEntry
// 	return entry
// }

func Virt2Phys(virtAddr string, pid uint64) string {
	virt, prefixFound := strings.CutPrefix(virtAddr, "0x")
	if !prefixFound {
		panic("Virtual address should start with 0x")
	}
	virtAsInt, err := strconv.ParseUint(virt, 16, 64)
	if err != nil {
		panic(err)
	}
	phys := uint64(C.virt_2_phys(unsafe.Pointer(uintptr(virtAsInt)), C.size_t(pid)))
	return fmt.Sprintf("0x%x", phys)
}

// Tell PTEditor to use kernel implementation
func PteditKernelImpl() {
	C.ptedit_use_implementation(C.PTEDIT_IMPL_KERNEL)
}

// Initialize PTEditor
func PteditInit() int {
	return int(C.ptedit_init())
}

// PTEditor clean up before exiting
func PteditCleanup() {
	C.ptedit_cleanup()
}

// func GetRootPhysAddr(pid uint64) uintptr {
// 	return uintptr(C.ptedit_get_paging_root(C.int(pid)))
// }

// func GetSystemPageSize() uint64 {
// 	return uint64(C.ptedit_get_pagesize())
// }

// Parse entry from uint64 into PTEntry type
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
		e.Color = "green-300"
	} else {
		e.Color = "red-300"
	}
	return e
}

func bytesToHex(data []byte) string {
	var hexString string
	for i := 0; i < len(data); i++ {
		hexString += fmt.Sprintf("%02X", data[i])
	}
	return hexString
}

// Traverse all page table levels and return mapped phys. pages
func GetAllPhysPages(pid uint64) []Page {
	var physPages []Page
	tableSize := 512
	pml4Entries := C.get_mapped_PML4_entries(C.size_t(pid))
	defer C.free(unsafe.Pointer(pml4Entries))
	pml4Size := int(C.FIRST_LEVEL_ENTRIES)
	pml4GoEntries := (*[1 << 30]C.PTEntry)(unsafe.Pointer(pml4Entries))[:pml4Size:pml4Size]
	for pml4i, pml4e := range pml4GoEntries {
		if pml4e.entry != 0 {
			pdptEntries := C.get_mapped_PDPT_entries(C.size_t(pid), C.size_t(pml4i))
			defer C.free(unsafe.Pointer(pdptEntries))
			pdptGoEntries := (*[1 << 30]C.PTEntry)(unsafe.Pointer(pdptEntries))[:tableSize:tableSize]
			for pdpti, pdpte := range pdptGoEntries {
				if pdpte.entry != 0 {
					pdEntries := C.get_mapped_PD_entries(C.size_t(pid), C.size_t(pml4i), C.size_t(pdpti))
					defer C.free(unsafe.Pointer(pdEntries))
					pdGoEntries := (*[1 << 30]C.PTEntry)(unsafe.Pointer(pdEntries))[:tableSize:tableSize]
					for pdi, pde := range pdGoEntries {
						if pde.entry != 0 {
							pteEntries := C.get_PTE_entries(C.size_t(pid), C.size_t(pml4i), C.size_t(pdpti), C.size_t(pdi))
							defer C.free(unsafe.Pointer(pteEntries))
							pteGoEntries := (*[1 << 30]C.PTEntry)(unsafe.Pointer(pteEntries))[:tableSize:tableSize]
							for ptei, ptee := range pteGoEntries {
								if ptee.entry != 0 {
									e := ParsePTEntry(uint64(ptee.entry), uint64(ptee.vaddr))
									pageSize := C.size_t(C.ptedit_get_pagesize())
									page := (*C.char)(C.malloc(pageSize))
									defer C.free(unsafe.Pointer(page))
									physPage := ReadPhysPage((uint64(ptee.entry) >> 12) & uint64((uint64(1)<<40)-1))
									p := Page{
										Content:     bytesToHex(physPage),
										Vfn:         e.Vfn,
										Translation: Translation{Pml4: fmt.Sprintf("%d", pml4i), Pdpt: fmt.Sprintf("%d", pdpti), Pd: fmt.Sprintf("%d", pdi), Pte: fmt.Sprintf("%d", ptei)},
									}
									physPages = append(physPages, p)
								}
							}
						}
					}
				}
			}
		}
	}
	return physPages
}

func CreateJSONFile(pages []Page, filename string, file *os.File) error {
	pageMap := make(map[string]interface{})

	// Populate the map with the data from the slice
	for idx, page := range pages {
		key := fmt.Sprintf("page-%d", idx)

		pageMap[key] = map[string]interface{}{
			"content":     page.Content,
			"vfn":         page.Vfn,
			"translation": page.Translation,
		}
	}

	encoder := json.NewEncoder(file)
	err := encoder.Encode(pageMap)
	if err != nil {
		return fmt.Errorf("Failed to encode JSON data")
	}
	return nil
}

func GetFirstLvl(pid uint64) []PTEntry {
	var ptEntries []PTEntry
	entries := C.get_mapped_PML4_entries(C.size_t(pid))
	defer C.free(unsafe.Pointer(entries))
	length := int(C.FIRST_LEVEL_ENTRIES)
	// Magic conversion from C entries into parsable Go entries
	goEntries := (*[1 << 30]C.PTEntry)(unsafe.Pointer(entries))[:length:length]
	for _, v := range goEntries {
		if v.entry != 0 {
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
	for _, v := range goEntries {
		if v.entry != 0 {
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
	for _, v := range goEntries {
		if v.entry != 0 {
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
	for _, v := range goEntries {
		if v.entry != 0 {
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

// func numOfEntriesPerLvl() int {
// 	return int(C.num_entries_per_lvl())
// }

func UpdateEntry(entryValues map[string]interface{}, pid uint64) (PTEntry, error) {
	// Type assertion as string
	pfn, ok := entryValues["pfn"].(string)
	if !ok {
		return PTEntry{}, fmt.Errorf("pfn is of the wrong type")
	}
	vfn, ok := entryValues["vfn"].(string)
	if !ok {
		return PTEntry{}, fmt.Errorf("vfn is of the wrong type")
	}
	vfn, found := strings.CutPrefix(vfn, "0x")
	if !found {
		return PTEntry{}, fmt.Errorf("Virt. address doesn't have a prefix")
	}
	vfn_int, err := strconv.ParseUint(vfn, 16, 64)
	if err != nil {
		return PTEntry{}, fmt.Errorf("Couln't parse virt. address to unsiged int")
	}
	vfn_ptr := uintptr(vfn_int)
	cEntry := C.ptedit_resolve_kernel(unsafe.Pointer(vfn_ptr), C.int(pid))
	entry := uint64(cEntry.pte)
	e := ParsePTEntry(entry, vfn_int)
	if uint64(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_PRESENT)) != entryValues["p"] {
		e.P = !e.P
		e = e.toggleColor()
		entry ^= (uint64(1) << uint64(C.PTEDIT_PAGE_BIT_PRESENT))
	}
	if uint64(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_RW)) != entryValues["w"] {
		e.W = !e.W
		entry ^= (uint64(1) << uint64(C.PTEDIT_PAGE_BIT_RW))
	}
	if uint64(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_USER)) != entryValues["u"] {
		e.U = !e.U
		entry ^= (uint64(1) << uint64(C.PTEDIT_PAGE_BIT_USER))
	}
	if uint64(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_PWT)) != entryValues["wt"] {
		e.Wt = !e.Wt
		entry ^= (uint64(1) << uint64(C.PTEDIT_PAGE_BIT_PWT))
	}
	if uint64(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_PCD)) != entryValues["dc"] {
		e.Dc = !e.Dc
		entry ^= (uint64(1) << uint64(C.PTEDIT_PAGE_BIT_PCD))
	}
	if uint64(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_ACCESSED)) != entryValues["a"] {
		e.A = !e.A
		entry ^= (uint64(1) << uint64(C.PTEDIT_PAGE_BIT_ACCESSED))
	}
	if uint64(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_DIRTY)) != entryValues["d"] {
		e.D = !e.D
		entry ^= (uint64(1) << uint64(C.PTEDIT_PAGE_BIT_DIRTY))
	}
	if uint64(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_PSE)) != entryValues["pat"] {
		e.Pat = !e.Pat
		entry ^= (uint64(1) << uint64(C.PTEDIT_PAGE_BIT_PSE))
	}
	if uint64(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_GLOBAL)) != entryValues["g"] {
		e.G = !e.G
		entry ^= (uint64(1) << uint64(C.PTEDIT_PAGE_BIT_GLOBAL))
	}
	if uint64(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_SOFTW1)) != entryValues["s1"] {
		e.S1 = !e.S1
		entry ^= (uint64(1) << uint64(C.PTEDIT_PAGE_BIT_SOFTW1))
	}
	if uint64(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_SOFTW2)) != entryValues["s2"] {
		e.S2 = !e.S2
		entry ^= (uint64(1) << uint64(C.PTEDIT_PAGE_BIT_SOFTW2))
	}
	if uint64(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_SOFTW3)) != entryValues["s3"] {
		e.S3 = !e.S3
		entry ^= (uint64(1) << uint64(C.PTEDIT_PAGE_BIT_SOFTW3))
	}
	if uint64(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_PAT_LARGE)) != entryValues["patl"] {
		e.PatL = !e.PatL
		entry ^= (uint64(1) << uint64(C.PTEDIT_PAGE_BIT_PAT_LARGE))
	}
	if uint64(C.bit_set(C.size_t(entry), C.PTEDIT_PAGE_BIT_NX)) != entryValues["nx"] {
		e.Nx = !e.Nx
		entry ^= (uint64(1) << uint64(C.PTEDIT_PAGE_BIT_NX))
	}
	if e.Pfn != pfn {
		pfnNoPrefix, found := strings.CutPrefix(pfn, "0x")
		if !found {
			return PTEntry{}, fmt.Errorf("Phys. address doesn't have a prefix")
		}
		pfn_int, err := strconv.ParseUint(pfnNoPrefix, 10, 64)
		if err != nil {
			return PTEntry{}, fmt.Errorf("Couln't parse phys. address to unsiged int")
		}
		e.Pfn = pfn
		entry = replacePfn(entry, pfn_int)
	}
	cEntry.pte = C.size_t(entry)
	C.ptedit_update_kernel(unsafe.Pointer(vfn_ptr), C.int(pid), &cEntry)
	// Evict old translation
	C.ptedit_invalidate_tlb(unsafe.Pointer(vfn_ptr))
	// Load new entry into TLB (might not be useful)
	cEntry = C.ptedit_resolve_kernel(unsafe.Pointer(vfn_ptr), C.int(pid))
	return e, nil
}

func ReadPhysPage(pfn uint64) []byte {
	pageSize := C.size_t(C.ptedit_get_pagesize())
	page := (*C.char)(C.malloc(pageSize))
	defer C.free(unsafe.Pointer(page))
	C.ptedit_read_physical_page(C.size_t(pfn), page)
	goPage := C.GoBytes(unsafe.Pointer(page), C.int(pageSize))
	return goPage
}

func WritePhysPage(pfn uint64, data []byte) {
	goDataAsStr := string(data)
	pageData := C.CString(goDataAsStr)
	defer C.free(unsafe.Pointer(pageData))
	C.ptedit_write_physical_page(C.size_t(pfn), pageData)
}

func ConvertHexStringsToBytes(hexStrings []string) ([]byte, error) {
	data := make([]byte, len(hexStrings))
	for i, e := range hexStrings {
		var b byte
		_, err := fmt.Sscanf(e, "%x", &b)
		if err != nil {
			fmt.Println("Error: ", err)
			return nil, err
		}
		data[i] = b
	}
	return data, nil
}

// Parses code sections (output of objdump)
func parseDisassembly(data string) ([]CodeSection, error) {
	var sections []CodeSection

	scanner := bufio.NewScanner(strings.NewReader(data))
	var currentSection CodeSection
	var codeBuilder strings.Builder
	inCodeBlock := false

	// Regular expressions to match the lines
	sectionRegex := regexp.MustCompile(`^000000000000[0-9a-f]{4} <(.+?)>:\s*$`)
	codeRegex := regexp.MustCompile(`^(\s*\w+:\s+.+)`)

	for scanner.Scan() {
		line := scanner.Text()

		// Match section header
		if sectionRegex.MatchString(line) {
			if inCodeBlock { // If we were in a code block, save the previous section
				currentSection.Code = codeBuilder.String()
				sections = append(sections, currentSection)
				codeBuilder.Reset()
			}

			// Extract section name and offset
			matches := sectionRegex.FindStringSubmatch(line)
			if len(matches) > 1 {
				currentSection.Name = matches[1]
				offset, err := strconv.ParseUint(line[:16], 16, 64) // First 16 characters for the offset
				if err != nil {
					return nil, fmt.Errorf("Couldn't parse the offsets of the code")
				}
				currentSection.Offset = offset
				inCodeBlock = true
			}
			continue
		}

		// Match code lines
		if inCodeBlock && codeRegex.MatchString(line) {
			codeBuilder.WriteString(line + "\n")
		}
	}

	// Save the last section if exists
	if inCodeBlock {
		currentSection.Code = codeBuilder.String()
		sections = append(sections, currentSection)
	}

	return sections, nil
}

func GetProgPath(pid uint64) (string, error) {
	path := fmt.Sprintf("/proc/%d/exe", pid)
	cmd := exec.Command("readlink", path)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("Failed to execute readlink on %s", path)
	}
	return (strings.ReplaceAll(string(output), "\n", "")), nil
}

func ParseProgramCode(pid uint64) ([]CodeSection, error) {
	progPath, err := GetProgPath(pid)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("objdump", "-d", "-M", "intel", progPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Failed to execute \"%s\".\nCheck if objdump exists on your system!", cmd)
	}
	codeSections, err := parseDisassembly(string(output))
	if err != nil {
		return nil, err
	}
	return codeSections, nil
}

func GetProgramBaseAddr(pid uint64) (string, error) {
	file, err := os.Open(fmt.Sprintf("/proc/%d/maps", pid))
	if err != nil {
		return "", fmt.Errorf("Couldn't open file %s", file.Name())
	}
	reader := bufio.NewReader(file)
	virtAddr, err := reader.ReadString('-')
	return fmt.Sprintf("0x%s", virtAddr[:len(virtAddr)-1]), nil
}
