package main

import (
	"cgo_test/handlers"
	"cgo_test/utils"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	var port string
	if len(os.Args) < 2 {
		port = "8000"
	} else {
		port = os.Args[1]
	}
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	utils.PteditKernelImpl()
	ind := utils.PteditInit()
	if ind == 0 {
		fmt.Printf("localhost:%s\n", port)
	} else {
		fmt.Println("Could not initialize ptedit (did you load the kernel module?)")
	}

	http.HandleFunc("/", handlers.MainPageHandler)
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "./static/img/hack.png") })
	http.HandleFunc("/pid", handlers.PidHandler)
	http.HandleFunc("/translate", handlers.Virt2PhysHandler)
	http.HandleFunc("/table", handlers.GeneralHandler)
	http.HandleFunc("/full-entry", handlers.FullEntryHandler)
	http.HandleFunc("/only-present", handlers.GeneralHandler)
	http.HandleFunc("/show-path", handlers.ShowPathHandler)
	http.HandleFunc("/save-entry", handlers.SaveEntryHandler)
	http.HandleFunc("/show-phys-page", handlers.ShowPhysPageHandler)
	http.HandleFunc("/download-phys-page", handlers.DownloadPhysPageHandler)
	http.HandleFunc("/upload-phys-page", handlers.UploadPhysPageHandler)
	http.HandleFunc("/save-phys-page", handlers.SavePhysPageHandler)
	http.HandleFunc("/close-modal", handlers.CloseModalHandler)
	http.HandleFunc("/close-info-modal", handlers.CloseInfoModalHandler)
	http.HandleFunc("/dump-pages", handlers.DumpPhysPagesHandler)
	http.HandleFunc("/show-info-modal", handlers.ShowInfoModalHandler)
	http.HandleFunc("/show-maps", handlers.ShowProcessMapsHandler)
	http.HandleFunc("/show-sections", handlers.ShowBinarySectionsHandler)
	http.HandleFunc("/attach", handlers.AttachGDBHandler)
	http.HandleFunc("/send-cmd", handlers.SendCommandHandler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))

	utils.PteditCleanup()
}
