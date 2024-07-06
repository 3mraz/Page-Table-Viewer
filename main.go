package main

import (
	"cgo_test/utils"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
)

var (
	pid           uint64 = 0
	pml4i         int64  = -1
	pdpti         int64  = -1
	pdi           int64  = -1
	ptei          int64  = -1
	ALL_ENTRIES          = make(map[string]utils.PTEntry)
	only_present1 bool   = false
	only_present2 bool   = false
	only_present3 bool   = false
	only_present4 bool   = false
)

func reset_env() {
	pid = 0
	pml4i = -1
	pdpti = -1
	pdi = -1
	ptei = -1
	ALL_ENTRIES = make(map[string]utils.PTEntry)
	only_present1 = false
	only_present2 = false
	only_present3 = false
	only_present4 = false
}

func setup_temp(idx int64, lvl uint64, present string) ([]utils.PTEntry, string, string, string, uint16, bool, error) {
	var entries []utils.PTEntry
	var tmplName string
	var nxtLvlName string
	var tmplPath string
	var numEntries uint16 = 512
	var only_present bool = false
	var err error = nil

	if present != "" {
		only_present = true
	}

	switch lvl + 1 {
	case 1:
		entries = utils.GetFirstLvl(pid)
		numEntries = 256
		tmplName = "pml4"
		nxtLvlName = "pdpt"
		tmplPath = "templates/pml4.html"
		only_present = !only_present1 && only_present
		only_present1 = only_present
	case 2:
		if idx != -1 {
			pml4i = idx
		}
		entries = utils.GetSecondLvl(pid, pml4i)
		tmplName = "pdpt"
		nxtLvlName = "pd"
		tmplPath = "templates/pdpt.html"
		only_present = !only_present2 && only_present
		only_present2 = only_present
	case 3:
		if idx != -1 {
			pdpti = idx
		}
		entries = utils.GetThirdLvl(pid, pml4i, pdpti)
		tmplName = "pd"
		nxtLvlName = "pte"
		tmplPath = "templates/pd.html"
		only_present = !only_present3 && only_present
		only_present3 = only_present
	case 4:
		if idx != -1 {
			pdi = idx
		}
		entries = utils.GetFourthLvl(pid, pml4i, pdpti, pdi)
		tmplName = "pte"
		nxtLvlName = "phys"
		tmplPath = "templates/pte.html"
		only_present = !only_present4 && only_present
		only_present4 = only_present
	default:
		err = errors.New("Couldn't setup template")
	}
	return entries, tmplName, nxtLvlName, tmplPath, numEntries, only_present, err
}

func _calc_vfn(lvl uint64, idx int64) string {
	switch lvl {
	case 0:
		return ("0x" + strconv.FormatInt(idx<<39, 16))
	case 1:
		return ("0x" + strconv.FormatInt(pml4i<<39|idx<<30, 16))
	case 2:
		return ("0x" + strconv.FormatInt(pml4i<<39|pdpti<<30|idx<<21, 16))
	case 3:
		return ("0x" + strconv.FormatInt(pml4i<<39|pdpti<<30|pdi<<21|idx<<12, 16))
	default:
		return ""
	}
}

func main() {
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	utils.PteditKernelImpl()
	ind := utils.PteditInit()
	if ind == 0 {
		fmt.Println("ptedit initialized")
	} else {
		fmt.Println("Could not initialize ptedit (did you load the kernel module?)")
	}

	// main page
	mainPageHandler := func(w http.ResponseWriter, r *http.Request) {
		reset_env()
		context := make(map[string]interface{})
		context["Pid"] = strconv.FormatUint(pid, 10)
		context["Phys"] = "0x000000000"
		templ := template.Must(template.ParseFiles("templates/index.html"))
		templ.Execute(w, context)
	}

	pidHandler := func(w http.ResponseWriter, r *http.Request) {
		// Ensure form data is parsed
		if err := r.ParseForm(); err != nil {
			http.Error(w, "cannot parse form", http.StatusBadRequest)
			return
		}

		// Parse the pid from the form value
		pId, err := strconv.ParseUint(r.PostFormValue("pid"), 10, 64)
		if err != nil {
			http.Error(w, "cannot parse process Id", http.StatusBadRequest)
			return
		}
		if pid != pId {
			pid = pId
			ALL_ENTRIES = make(map[string]utils.PTEntry)
		}

		pidHTML := fmt.Sprintf("<label id='pid' for='virt'><b>Process Id:</b> %d</label>", pid)

		tmpl, _ := template.New("pidHTML").Parse(pidHTML)
		if err := tmpl.Execute(w, nil); err != nil {
			http.Error(w, "cannot execute template", http.StatusInternalServerError)
		}
	}

	Virt2PhysHandler := func(w http.ResponseWriter, r *http.Request) {
		virtAddr := r.PostFormValue("virt")
		phys := utils.Virt2Phys(virtAddr, pid)
		context := make(map[string]interface{})
		physHTML := fmt.Sprintf("<label id='phys' for='virt'>%s</label>", phys)
		tmpl, _ := template.New("physHTML").Parse(physHTML)
		if err := tmpl.Execute(w, context); err != nil {
			http.Error(w, "cannot execute template", http.StatusInternalServerError)
		}
	}

	generalHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "invalid method", http.StatusBadRequest)
			return
		}
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Couldn't parse the form", http.StatusBadRequest)
		}
		lvl, err := strconv.ParseUint(r.FormValue("lvl"), 10, 64)
		if err != nil {
			fmt.Println("error parsing lvl")
			return
		}
		idx, err := strconv.ParseInt(r.FormValue("idx"), 10, 64)
		if err != nil {
			fmt.Println("error parsing idx")
			return
		}
		present := r.FormValue("present")
		entries, tmplName, nxtLvlName, tmplPath, numEntries, only_present, err := setup_temp(idx, lvl, present)
		if err != nil {
			http.Error(w, "Couldn't setup table template", http.StatusBadRequest)
			return
		}
		vAddrs := make(map[uint16]interface{})
		for idx, entry := range entries {
			ALL_ENTRIES[entry.Pfn] = entry
			for i := uint16(0); i < numEntries; i++ {
				vfn := _calc_vfn(lvl, int64(i))
				if vfn == entry.Vfn {
					vAddrs[i] = entry
					if idx != 0 {
						break
					}
				} else if idx == 0 {
					vAddrs[i] = utils.PTEntry{Vfn: vfn}
				}
			}
		}
		context := make(map[string]interface{})
		context["vAddrs"] = vAddrs
		context["onlyPresent"] = only_present
		context["tmplName"] = tmplName
		context["nxtLvlName"] = nxtLvlName
		context["lvl"] = lvl + 1
		tmpl := template.Must(template.ParseFiles(
			tmplPath,
			"templates/pte-table.html",
		))
		tmpl.ExecuteTemplate(w, tmplName, context)
	}

	fullEntryHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Not a valid method", http.StatusBadRequest)
			return
		}
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "couldn't parse form", http.StatusBadRequest)
			return
		}
		entryPfn := r.FormValue("pfn")
		context := make(map[string]utils.PTEntry)
		context["entry"] = ALL_ENTRIES[entryPfn]
		tmpl := template.Must(template.ParseFiles("templates/full-entry.html"))
		tmpl.ExecuteTemplate(w, "full-entry", context)
	}

	http.HandleFunc("/", mainPageHandler)
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "./static/img/hack.png") })
	http.HandleFunc("/pid", pidHandler)
	http.HandleFunc("/translate", Virt2PhysHandler)
	http.HandleFunc("/table", generalHandler)
	http.HandleFunc("/full-entry", fullEntryHandler)
	http.HandleFunc("/only-present", generalHandler)

	log.Fatal(http.ListenAndServe(":8000", nil))

	// addr := C.mmap(C.NULL, C.size_t(4096), C.PROT_READ|C.PROT_WRITE, C.MAP_PRIVATE|C.MAP_ANONYMOUS, C.int(-1), C.long(0))
	// if addr == unsafe.Pointer(uintptr(0)) {
	// 	fmt.Println("mmap failed")
	// 	return
	// } else {
	// 	fmt.Println(addr)
	// }
	//
	// C.memset(addr, C.int(int('B')), C.size_t(4096))
	// vm := init_Entry(C.ptedit_resolve_kernel(addr, C.int(0)))
	// fmt.Printf("vaddr: %p\n", unsafe.Pointer(vm.vaddr))

	utils.PteditCleanup()
}
