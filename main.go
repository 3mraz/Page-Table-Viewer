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
	pid   uint64 = 0
	pml4i uint64 = 0
	pdpti uint64 = 0
	pdi   uint64 = 0
)

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
	h1 := func(w http.ResponseWriter, r *http.Request) {
		context := make(map[string]any)
		context["Pid"] = strconv.FormatUint(pid, 10)
		context["Phys"] = "0x000000000"
		tmpl := template.Must(template.ParseFiles("templates/index.html"))
		tmpl.Execute(w, context)
	}

	// get pml4 table
	h2 := func(w http.ResponseWriter, r *http.Request) {
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
		pid = pId

		pidHTML := fmt.Sprintf("<label id='pid' for='virt'><b>Process Id:</b> %d</label>", pid)

		tmpl, _ := template.New("pidHTML").Parse(pidHTML)
		if err := tmpl.Execute(w, nil); err != nil {
			http.Error(w, "cannot execute template", http.StatusInternalServerError)
		}
	}

	h3 := func(w http.ResponseWriter, r *http.Request) {
		virtAddr := r.PostFormValue("virt")
		phys := utils.Virt2Phys(virtAddr, pid)
		context := make(map[string]any)
		physHTML := fmt.Sprintf("<label id='phys' for='virt'>%s</label>", phys)
		tmpl, _ := template.New("physHTML").Parse(physHTML)
		if err := tmpl.Execute(w, context); err != nil {
			http.Error(w, "cannot execute template", http.StatusInternalServerError)
		}
	}

	h4 := func(w http.ResponseWriter, r *http.Request) {
		vAddrs := make(map[int]utils.PTEntry)
		pml4Entries := utils.GetFirstLvl(pid)
		for idx, entry := range pml4Entries {
			for i := 0; i < 256; i++ {
				if ("0x" + strconv.FormatUint(uint64(i)<<39, 16)) == entry.Vfn {
					vAddrs[i] = entry
					if idx != 0 {
						break
					}
				} else {
					if idx == 0 {
						vAddrs[i] = utils.PTEntry{Vfn: "0x" + strconv.FormatUint(uint64(i)<<39, 16)}
					}
				}
			}
		}
		context := make(map[string]any)
		context["vAddrs"] = vAddrs
		tmpl := template.Must(template.ParseFiles("templates/index.html", "templates/pml4.html"))
		tmpl.ExecuteTemplate(w, "pml4", context)
	}

	h5 := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			return
		}
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "couldn't parse form", http.StatusBadRequest)
			return
		}
		pml4_idx, err := strconv.ParseUint(r.FormValue("pml4_idx"), 10, 64)
		pml4i = pml4_idx
		if err != nil {
			http.Error(w, "Error parsing idx", http.StatusBadRequest)
			return
		}
		vAddrs := make(map[int]utils.PTEntry)
		pdptEntries := utils.GetSecondLvl(pid, pml4i)
		for idx1, entry := range pdptEntries {
			for i := 0; i < 512; i++ {
				if ("0x" + strconv.FormatUint((uint64(pml4i)<<39)|(uint64(i)<<30), 16)) == entry.Vfn {
					vAddrs[i] = entry
					if idx1 != 0 {
						break
					}
				} else {
					if idx1 == 0 {
						vAddrs[i] = utils.PTEntry{Vfn: "0x" + strconv.FormatUint((uint64(pml4i)<<39)|(uint64(i)<<30), 16)}
					}
				}
			}
		}
		context := make(map[string]any)
		context["vAddrs"] = vAddrs
		tmpl := template.Must(template.ParseFiles("templates/index.html", "templates/pml4.html", "templates/pdpt.html"))
		tmpl.ExecuteTemplate(w, "pdpt", context)
	}

	h6 := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			return
		}
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "couldn't parse form", http.StatusBadRequest)
			return
		}
		pdpt_idx, err := strconv.ParseUint(r.FormValue("pdpt_idx"), 10, 64)
		pdpti = pdpt_idx
		if err != nil {
			http.Error(w, "Error parsing idx", http.StatusBadRequest)
			return
		}
		vAddrs := make(map[int]utils.PTEntry)
		pdEntries := utils.GetThirdLvl(pid, pml4i, pdpti)
		for idx1, entry := range pdEntries {
			for i := 0; i < 512; i++ {
				if ("0x" + strconv.FormatUint((uint64(pml4i)<<39)|(uint64(pdpti)<<30)|(uint64(i)<<21), 16)) == entry.Vfn {
					vAddrs[i] = entry
					if idx1 != 0 {
						break
					}
				} else {
					if idx1 == 0 {
						vAddrs[i] = utils.PTEntry{Vfn: "0x" + strconv.FormatUint((uint64(pml4i)<<39)|(uint64(pdpti)<<30)|(uint64(i)<<21), 16)}
					}
				}
			}
		}
		context := make(map[string]any)
		context["vAddrs"] = vAddrs
		tmpl := template.Must(template.ParseFiles(
			"templates/index.html",
			"templates/pml4.html",
			"templates/pdpt.html",
			"templates/pd.html",
		))
		tmpl.ExecuteTemplate(w, "pd", context)
	}

	h7 := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			return
		}
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "couldn't parse form", http.StatusBadRequest)
			return
		}
		pd_idx, err := strconv.ParseUint(r.FormValue("pd_idx"), 10, 64)
		pdi = pd_idx
		if err != nil {
			http.Error(w, "Error parsing idx", http.StatusBadRequest)
			return
		}
		vAddrs := make(map[int]utils.PTEntry)
		pteEntries := utils.GetFourthLvl(pid, pml4i, pdpti, pdi)
		for idx1, entry := range pteEntries {
			for i := 0; i < 512; i++ {
				if ("0x" + strconv.FormatUint((uint64(pml4i)<<39)|(uint64(pdpti)<<30)|(uint64(pdi)<<21)|(uint64(i)<<12), 16)) == entry.Vfn {
					vAddrs[i] = entry
					if idx1 != 0 {
						break
					}
				} else {
					if idx1 == 0 {
						vAddrs[i] = utils.PTEntry{Vfn: "0x" + strconv.FormatUint((uint64(pml4i)<<39)|(uint64(pdpti)<<30)|(uint64(pdi)<<21)|(uint64(i)<<12), 16)}
					}
				}
			}
		}
		context := make(map[string]any)
		context["vAddrs"] = vAddrs
		tmpl := template.Must(template.ParseFiles(
			"templates/index.html",
			"templates/pml4.html",
			"templates/pdpt.html",
			"templates/pd.html",
			"templates/pte.html",
		))
		tmpl.ExecuteTemplate(w, "pte", context)
	}

	http.HandleFunc("/", h1)
	http.HandleFunc("/pid", h2)
	http.HandleFunc("/translate", h3)
	http.HandleFunc("/show-tables", h4)
	http.HandleFunc("/second-lvl", h5)
	http.HandleFunc("/third-lvl", h6)
	http.HandleFunc("/fourth-lvl", h7)

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
