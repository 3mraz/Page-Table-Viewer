package main

import (
	"cgo_test/utils"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
)

var pid uint64

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

	h1 := func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("templates/index.html"))
		tmpl.Execute(w, nil)
	}

	h3 := func(w http.ResponseWriter, r *http.Request) {
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
		utils.GetFirstLvl(pid)

		var context (struct {
			Pid  string
			Phys string
		})
		context.Pid = strconv.FormatUint(pid, 10)
		context.Phys = "0x000000000"
		// Load and execute the template
		tmpl := template.Must(template.ParseFiles("templates/virt2phys.html"))
		if err := tmpl.Execute(w, context); err != nil {
			http.Error(w, "cannot execute template", http.StatusInternalServerError)
		}
	}

	h2 := func(w http.ResponseWriter, r *http.Request) {
		virtAddr := r.PostFormValue("virt")
		phys := utils.Virt2Phys(virtAddr, pid)
		tmpl := template.Must(template.ParseFiles("templates/physical.html"))
		context := struct {
			Phys string
		}{
			Phys: phys,
		}
		if err := tmpl.Execute(w, context); err != nil {
			http.Error(w, "cannot execute template", http.StatusInternalServerError)
		}
	}

	http.HandleFunc("/", h1)
	http.HandleFunc("/translate", h2)
	http.HandleFunc("/pid", h3)

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
