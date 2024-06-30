package main

import (
	"cgo_test/utils"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
)

var (
	pid         uint64 = 0
	pml4i       uint64 = 0
	pdpti       uint64 = 0
	pdi         uint64 = 0
	ptei        uint64 = 0
	ALL_ENTRIES        = make(map[string]utils.PTEntry)
)

func _calc_vfn(lvl uint64, idx uint64) string {
	switch lvl {
	case 0:
		return ("0x" + strconv.FormatUint(idx<<39, 16))
	case 1:
		return ("0x" + strconv.FormatUint(pml4i<<39|idx<<30, 16))
	case 2:
		return ("0x" + strconv.FormatUint(pml4i<<39|pdpti<<30|idx<<21, 16))
	case 3:
		return ("0x" + strconv.FormatUint(pml4i<<39|pdpti<<30|pdi<<21|idx<<12, 16))
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
		ALL_ENTRIES = make(map[string]utils.PTEntry)
		pid = 0
		context := make(map[string]any)
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
		context := make(map[string]any)
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
		idx, err := strconv.ParseUint(r.FormValue("idx"), 10, 64)
		if err != nil {
			http.Error(w, "Couldn't parse request values", http.StatusBadRequest)
		}
		var entries []utils.PTEntry
		var tmplName string
		var nxtLvlName string
		var tmplPath string
		numEntries := 512
		switch lvl + 1 {
		case 1:
			entries = utils.GetFirstLvl(pid)
			numEntries = 256
			tmplName = "pml4"
			nxtLvlName = "pdpt"
			tmplPath = "templates/pml4.html"
		case 2:
			pml4i = idx
			entries = utils.GetSecondLvl(pid, pml4i)
			tmplName = "pdpt"
			nxtLvlName = "pd"
			tmplPath = "templates/pdpt.html"
		case 3:
			pdpti = idx
			entries = utils.GetThirdLvl(pid, pml4i, pdpti)
			tmplName = "pd"
			nxtLvlName = "pte"
			tmplPath = "templates/pd.html"
		case 4:
			pdi = idx
			entries = utils.GetFourthLvl(pid, pml4i, pdpti, pdi)
			tmplName = "pte"
			nxtLvlName = "phys"
			tmplPath = "templates/pte.html"
		default:
			http.Error(w, "Wrong page table level", http.StatusBadRequest)
			return
		}
		vAddrs := make(map[int]utils.PTEntry)
		for idx, entry := range entries {
			ALL_ENTRIES[entry.Pfn] = entry
			for i := 0; i < numEntries; i++ {
				vfn := _calc_vfn(lvl, uint64(i))
				if vfn == entry.Vfn {
					vAddrs[i] = entry
					if idx != 0 {
						break
					}
				} else {
					if idx == 0 {
						vAddrs[i] = utils.PTEntry{Vfn: vfn}
					}
				}
			}
		}
		context := make(map[string]any)
		context["vAddrs"] = vAddrs
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
	http.HandleFunc("/pid", pidHandler)
	http.HandleFunc("/translate", Virt2PhysHandler)
	http.HandleFunc("/table", generalHandler)
	http.HandleFunc("/full-entry", fullEntryHandler)

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
