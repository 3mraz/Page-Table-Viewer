package handlers

import (
	"bufio"
	"cgo_test/utils"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	pid               uint64                              = 0
	pml4i             int64                               = -1
	pdpti             int64                               = -1
	pdi               int64                               = -1
	ptei              int64                               = -1
	only_present_pml4 bool                                = false
	only_present_pdpt bool                                = false
	only_present_pd   bool                                = false
	only_present_pte  bool                                = false
	cstate            map[string]map[uint16]utils.PTEntry = make(map[string]map[uint16]utils.PTEntry)
	warnings          []string
	currentMemPage    utils.MemPage
	baseAddr          string
)

func reset_env() {
	pid = 0
	pml4i = -1
	pdpti = -1
	pdi = -1
	ptei = -1
	only_present_pml4 = false
	only_present_pdpt = false
	only_present_pd = false
	only_present_pte = false
	cstate = make(map[string]map[uint16]utils.PTEntry)
	warnings = make([]string, 0)
	currentMemPage = utils.MemPage{}
	baseAddr = ""
}

// Returns the entries for a particular level
// exist: Entries already fetched
func getEntries(lvl uint64, exist string, lvlName string, numEntries uint16) (map[uint16]utils.PTEntry, error) {
	if exist != "" {
		return cstate[lvlName], nil
	} else {
		var entries []utils.PTEntry
		switch lvl {
		case 0:
			entries = utils.GetFirstLvl(pid)
		case 1:
			entries = utils.GetSecondLvl(pid, pml4i)
		case 2:
			entries = utils.GetThirdLvl(pid, pml4i, pdpti)
		case 3:
			entries = utils.GetFourthLvl(pid, pml4i, pdpti, pdi)
		default:
			return nil, fmt.Errorf("Couln't get entries")
		}
		// Prepare map of indecies to entries for the template
		vAddrs := make(map[uint16]utils.PTEntry)
		for index, entry := range entries {
			for i := uint16(0); i < numEntries; i++ {
				vfn := calc_vfn(lvl, int64(i))
				if vfn == entry.Vfn {
					vAddrs[i] = entry
					if index != 0 {
						break
					}
				} else if index == 0 {
					e := utils.PTEntry{Vfn: vfn, Color: "red-300", Pfn: "0x0"}
					vAddrs[i] = e
				}
			}
		}
		cstate[lvlName] = vAddrs
		return vAddrs, nil
	}
}

// Extract level indecies of the virt. address
func parseVirt(virtAddr string) (pml4i int64, pdpti int64, pdi int64, ptei int64, err error) {
	if strings.HasPrefix(virtAddr, "0x") {
		virtNum, err := strconv.ParseUint(virtAddr[2:], 16, 64)
		if err != nil {
			return -1, -1, -1, -1, fmt.Errorf("Couldn't parse virtAddr")
		}
		var pml4Idx int64 = int64(virtNum & (0x1ff << 39) >> 39)
		var pdptIdx int64 = int64(virtNum & (0x1ff << 30) >> 30)
		var pdIdx int64 = int64(virtNum & (0x1ff << 21) >> 21)
		var pteIdx int64 = int64(virtNum & (0x1ff << 12) >> 12)
		return pml4Idx, pdptIdx, pdIdx, pteIdx, nil
	}
	return -1, -1, -1, -1, fmt.Errorf("Couldn't parse virtAddr")
}

func basePlusOffset(offset uint64) (string, error) {
	baseInt, err := strconv.ParseUint(baseAddr[2:], 16, 64)
	if err != nil {
		return "", err
	}
	fullAddr := fmt.Sprintf("0x%s", strconv.FormatUint(baseInt+offset, 16))
	return fullAddr, nil
}

func calc_offset(vfn uint64) (uint64, error) {
	baseInt, err := strconv.ParseUint(baseAddr[2:], 16, 64)
	if err != nil {
		fmt.Println(err)
		return 0, err
	}
	fullOffset := vfn - baseInt
	return fullOffset, nil
}

func calc_vfn(lvl uint64, idx int64) string {
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

// Returns needed data to show a table
func setup_temp(idx int64, lvl uint64, present string) (map[uint16]utils.PTEntry, string, string, string, bool, error) {
	var tmplName string
	var nxtLvlName string
	var tmplPath string
	var numEntries uint16 = 512
	var only_present bool = true
	var err error = nil

	if present != "" {
		only_present = true
	}

	switch lvl {
	case 0:
		numEntries = 256
		tmplName = "pml4"
		nxtLvlName = "pdpt"
		tmplPath = "templates/pml4.html"
		only_present = !only_present_pml4 && only_present // alternate each click
		only_present_pml4 = only_present                  // update program state
	case 1:
		if idx != -1 {
			pml4i = idx
		}
		tmplName = "pdpt"
		nxtLvlName = "pd"
		tmplPath = "templates/pdpt.html"
		only_present = !only_present_pdpt && only_present
		only_present_pdpt = only_present
	case 2:
		if idx != -1 {
			pdpti = idx
		}
		tmplName = "pd"
		nxtLvlName = "pte"
		tmplPath = "templates/pd.html"
		only_present = !only_present_pd && only_present
		only_present_pd = only_present
	case 3:
		if idx != -1 {
			pdi = idx
		}
		tmplName = "pte"
		nxtLvlName = "phys"
		tmplPath = "templates/pte.html"
		only_present = !only_present_pte && only_present
		only_present_pte = only_present
	default:
		err = fmt.Errorf("Couldn't setup template")
	}
	entries, err := getEntries(lvl, present, tmplName, numEntries)
	if err != nil {
		fmt.Println(err)
		panic("couldn't retrieve entries")
	}
	return entries, tmplName, nxtLvlName, tmplPath, only_present, err
}

// main page
func MainPageHandler(w http.ResponseWriter, r *http.Request) {
	reset_env()
	context := make(map[string]interface{})
	context["Pid"] = strconv.FormatUint(pid, 10)
	context["Phys"] = "0x000000000"
	templ := template.Must(template.ParseFiles("templates/index.html"))
	templ.Execute(w, context)
}

func PidHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "cannot parse form", http.StatusBadRequest)
		return
	}

	// Parse pid
	pId, err := strconv.ParseUint(r.PostFormValue("pid"), 10, 64)
	if err != nil {
		http.Error(w, "cannot parse process Id", http.StatusBadRequest)
		return
	}
	if pid != pId {
		pid = pId
	}

	// Prepare to update the template
	pidHTML := fmt.Sprintf("<label id='pid' for='virt'><b>Process Id:</b> %d</label>", pid)

	// Set program base address
	baseAddr, err = utils.GetProgramBaseAddr(pid)
	if err != nil {
		http.Error(w, fmt.Sprintf("%s", err), http.StatusInternalServerError)
		return
	}

	tmpl, _ := template.New("pidHTML").Parse(pidHTML)
	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, "cannot execute template", http.StatusInternalServerError)
		return
	}
}

// Handles the translation functionality
func Virt2PhysHandler(w http.ResponseWriter, r *http.Request) {
	virtAddr := r.PostFormValue("virt")
	phys := ""
	if !(utils.ValidVirt(virtAddr)) {
		phys = "Invalid Syntax"
	} else {
		phys = utils.Virt2Phys(virtAddr, pid)
	}
	context := make(map[string]interface{})
	physHTML := fmt.Sprintf("<label id='phys' for='virt'>%s</label>", phys)
	tmpl, _ := template.New("physHTML").Parse(physHTML)
	if err := tmpl.Execute(w, context); err != nil {
		http.Error(w, "cannot execute template", http.StatusInternalServerError)
	}
}

func ShowPathHandler(w http.ResponseWriter, r *http.Request) {
	virt := r.PostFormValue("virt")
	if !utils.ValidVirt(virt) {
		tmpl := template.New("page-tables")
		tmpl.Parse(`
  <div id="page-tables" class="w-full h-screen mt-3 flex">
    {{block "pml4" .}}
    <div id="pml4"></div>
    {{end}} {{block "pdpt" .}}
    <div id="pdpt"></div>
    {{end}} {{block "pd" .}}
    <div id="pd"></div>
    {{end}} {{block "pte" .}}
    <div id="pte"></div>
    {{end}}
  </div>`)
		tmpl.ExecuteTemplate(w, "page-tables", nil)
		return
	}
	pml4Idx, pdptIdx, pdIdx, pteIdx, err := parseVirt(virt)
	if err != nil {
		fmt.Println(err)
		return
	}
	// Setup data for all levels
	pml4Enries, pml4TmplName, pml4NxtLvlName, pml4TmplPath, _, pml4_err := setup_temp(0, 0, "")
	pdptEnries, pdptTmplName, pdptNxtLvlName, pdptTmplPath, _, pdpt_err := setup_temp(pml4Idx, 1, "")
	pdEnries, pdTmplName, pdNxtLvlName, pdTmplPath, _, pd_err := setup_temp(pdptIdx, 2, "")
	pteEnries, pteTmplName, pteNxtLvlName, pteTmplPath, _, pte_err := setup_temp(pdIdx, 3, "")
	if pml4_err != nil || pdpt_err != nil || pd_err != nil || pte_err != nil {
		fmt.Println("Error while getting entries")
		return
	}
	// Highlight entries with blue
	e := pml4Enries[uint16(pml4Idx)]
	e.Color = "blue-400"
	pml4Enries[uint16(pml4Idx)] = e
	e = pdptEnries[uint16(pdptIdx)]
	e.Color = "blue-400"
	pdptEnries[uint16(pdptIdx)] = e
	e = pdEnries[uint16(pdIdx)]
	e.Color = "blue-400"
	pdEnries[uint16(pdIdx)] = e
	e = pteEnries[uint16(pteIdx)]
	e.Color = "blue-400"
	pteEnries[uint16(pteIdx)] = e
	context := make(map[string]map[string]interface{})

	context["pml4"] = make(map[string]interface{})
	context1 := context["pml4"]
	context1["entries"] = pml4Enries
	context1["onlyPresent"] = true
	only_present_pml4 = true
	context1["tmplName"] = pml4TmplName
	context1["nxtLvlName"] = pml4NxtLvlName
	context1["lvl"] = 1
	context["pml4"] = context1

	context["pdpt"] = make(map[string]interface{})
	context2 := context["pdpt"]
	context2["entries"] = pdptEnries
	context2["onlyPresent"] = true
	only_present_pdpt = true
	only_present_pd = true
	context2["tmplName"] = pdptTmplName
	context2["nxtLvlName"] = pdptNxtLvlName
	context2["lvl"] = 2
	context["pdpt"] = context2

	context["pd"] = make(map[string]interface{})
	context3 := context["pd"]
	context3["entries"] = pdEnries
	context3["onlyPresent"] = true
	only_present_pte = true
	context3["tmplName"] = pdTmplName
	context3["nxtLvlName"] = pdNxtLvlName
	context3["lvl"] = 3
	context["pd"] = context3

	context["pte"] = make(map[string]interface{})
	context4 := context["pte"]
	context4["entries"] = pteEnries
	context4["onlyPresent"] = true
	context4["tmplName"] = pteTmplName
	context4["nxtLvlName"] = pteNxtLvlName
	context4["lvl"] = 4
	context["pte"] = context4
	tmpl := template.Must(template.ParseFiles("templates/index.html", pml4TmplPath, pdptTmplPath, pdTmplPath, pteTmplPath, "templates/pte-table.html"))
	tmpl.ExecuteTemplate(w, "page-tables", context)
}

func GeneralHandler(w http.ResponseWriter, r *http.Request) {
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
	entries, tmplName, nxtLvlName, tmplPath, only_present, err := setup_temp(idx, lvl, present)
	if err != nil {
		http.Error(w, "Couldn't setup table template", http.StatusBadRequest)
		return
	}
	context := make(map[string]map[string]interface{})
	context[tmplName] = make(map[string]interface{})
	context[tmplName]["entries"] = entries
	context[tmplName]["onlyPresent"] = only_present
	context[tmplName]["tmplName"] = tmplName
	context[tmplName]["nxtLvlName"] = nxtLvlName
	context[tmplName]["lvl"] = lvl + 1
	tmpl := template.Must(template.ParseFiles(
		tmplPath,
		"templates/pte-table.html",
	))
	tmpl.ExecuteTemplate(w, tmplName, context)
}

// Handles showing the extended entry
func FullEntryHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "couldn't parse form", http.StatusBadRequest)
		return
	}
	entryVfn := r.PostFormValue("vfn")
	tableName := r.PostFormValue("tName")
	idx, err := strconv.ParseUint(r.PostFormValue("idx"), 10, 16)
	readView := r.PostFormValue("read")
	var e utils.PTEntry
	var ok bool
	if err != nil {
		http.Error(w, "Couldn't parse index of full entry", http.StatusBadRequest)
		ok = false
		e = utils.PTEntry{}
	} else {
		e, ok = cstate[tableName][uint16(idx)]
	}
	if !ok {
		e = utils.PTEntry{Vfn: entryVfn, Pfn: "0x0", Color: "red-300"}
	}

	context := make(map[string]interface{})
	context["entry"] = e
	context["readView"] = readView
	context["idx"] = idx
	context["tName"] = tableName
	context["warnings"] = warnings
	tmpl := template.Must(template.ParseFiles("templates/full-entry.html"))
	tmpl.ExecuteTemplate(w, "full-entry", context)
}

func SaveEntryHandler(wr http.ResponseWriter, r *http.Request) {
	var nx, p, patl, pat, s1, s2, s3, d, dc, u, a, w, wt, g uint64
	pfn := "0x0"
	warnings = make([]string, 0)
	p, _ = strconv.ParseUint(r.PostFormValue("p"), 10, 64)
	if p == 1 {
		nx, _ = strconv.ParseUint(r.PostFormValue("nx"), 10, 64)
		pat, _ = strconv.ParseUint(r.PostFormValue("pat"), 10, 64)
		patl, _ = strconv.ParseUint(r.PostFormValue("patl"), 10, 64)
		s1, _ = strconv.ParseUint(r.PostFormValue("s1"), 10, 64)
		s2, _ = strconv.ParseUint(r.PostFormValue("s2"), 10, 64)
		s3, _ = strconv.ParseUint(r.PostFormValue("s3"), 10, 64)
		u, _ = strconv.ParseUint(r.PostFormValue("u"), 10, 64)
		dc, _ = strconv.ParseUint(r.PostFormValue("dc"), 10, 64)
		d, _ = strconv.ParseUint(r.PostFormValue("d"), 10, 64)
		a, _ = strconv.ParseUint(r.PostFormValue("a"), 10, 64)
		w, _ = strconv.ParseUint(r.PostFormValue("w"), 10, 64)
		wt, _ = strconv.ParseUint(r.PostFormValue("wt"), 10, 64)
		g, _ = strconv.ParseUint(r.PostFormValue("g"), 10, 64)
		pfn = r.PostFormValue("pfn")
	}
	if !utils.ValidPhys(pfn) {
		warnings = append(warnings, "The PFN you entered is not valid.")
		return
	}
	vfn := r.PostFormValue("vfn")
	tableName := r.PostFormValue("tName")

	eValues := make(map[string]interface{})
	eValues["p"] = p
	eValues["pfn"] = pfn
	eValues["vfn"] = vfn
	eValues["nx"] = nx
	eValues["pat"] = pat
	eValues["patl"] = patl
	eValues["s1"] = s1
	eValues["s2"] = s2
	eValues["s3"] = s3
	eValues["u"] = u
	eValues["dc"] = dc
	eValues["d"] = d
	eValues["a"] = a
	eValues["w"] = w
	eValues["wt"] = wt
	eValues["g"] = g
	// Update entry in memory
	e, err := utils.UpdateEntry(eValues, pid)
	if err != nil {
		http.Error(wr, "Failed to update the entry", http.StatusBadRequest)
		return
	}

	idx, err := strconv.ParseUint(r.PostFormValue("idx"), 10, 16)
	if err != nil {
		http.Error(wr, "Couldn't parse index in save entry", http.StatusBadRequest)
		return
	}

	// Update entry in program state
	cstate[tableName][uint16(idx)] = e
}

func ShowPhysPageHandler(w http.ResponseWriter, r *http.Request) {
	t := r.PostFormValue("type")
	pageObtained := r.PostFormValue("obtained")
	context := make(map[string]interface{})
	// For string view of phys. page
	if t == "string" {
		pageString := string(currentMemPage.PageBytes)
		context["string"] = pageString
	} else if t == "hex" { // For hex view
		if pageObtained != "true" {
			pfn, err := strconv.ParseUint(r.PostFormValue("pfn")[2:], 16, 64)
			if err != nil {
				fmt.Println("Error parsing pfn")
				http.Error(w, "Couldn't parse pfn in ShowPhysPage", http.StatusBadRequest)
				return
			}
			vfn, err := strconv.ParseUint(r.PostFormValue("vfn")[2:], 16, 64)
			if err != nil {
				fmt.Println("Error parsing vfn")
				http.Error(w, "Couldn't parse vfn in ShowPhysPage", http.StatusBadRequest)
				return
			}
			// Update current viewing phys. page
			currentMemPage.Offset, err = calc_offset(vfn)
			if err != nil {
				fmt.Println("Couldn't parse the offset of the current page")
				http.Error(w, "Couldn't parse the offset of the current page", http.StatusInternalServerError)
				return
			}
			currentMemPage.Nx = r.PostFormValue("nx")
			currentMemPage.PageBytes = utils.ReadPhysPage(pfn)
			var i uint64
			for ; i < 256; i++ {
				currentMemPage.Addresses[i] = strconv.FormatUint(vfn+(i*uint64(16)), 16)
			}
		}

		// TODO: use dynamic size based on page size

		// Use 2 different arrays for better layout in the template
		var bytes1 [256][8]string
		var bytes2 [256][8]string
		for i := 0; i < 4096; i++ {
			if (i % 16) < 8 {
				bytes1[i/16][i%8] = fmt.Sprintf("%02x", currentMemPage.PageBytes[i])
			} else {
				bytes2[i/16][i%8] = fmt.Sprintf("%02x", currentMemPage.PageBytes[i])
			}
		}
		context["bytes1"] = bytes1
		context["bytes2"] = bytes2
		context["addresses"] = currentMemPage.Addresses
	} else if t == "code" { // For code view
		// Parse the output of objdump
		codeSections, err := utils.ParseProgramCode(pid)
		if err != nil {
			context["code"] = fmt.Sprintf("%s", err)
		} else {
			strBuilder := strings.Builder{}
			for _, section := range codeSections {
				// If section inside page boundaries, add it's fields to the template
				if (section.Offset >= currentMemPage.Offset) && (section.Offset < (currentMemPage.Offset + uint64(4096))) {
					addr, err := basePlusOffset(section.Offset)
					if err != nil {
						fmt.Println(err)
						http.Error(w, fmt.Sprintf("%s", err), http.StatusInternalServerError)
						return
					}
					strBuilder.WriteString(fmt.Sprintf("%s     %s\n%s\n\n\n", addr, section.Name, section.Code))
				}
			}
			context["code"] = strBuilder.String()
		}
	} else { // For edit view
		var b [256][16]string
		for i := 0; i < 4096; i++ {
			b[i/16][i%16] = fmt.Sprintf("%02x", currentMemPage.PageBytes[i])
		}
		context["bytes"] = b
		context["addresses"] = currentMemPage.Addresses
	}
	context["type"] = t
	context["nx"] = currentMemPage.Nx
	tmpl := template.Must(template.ParseFiles("templates/modal.html"))
	tmpl.ExecuteTemplate(w, "modal", context)
}

func SavePhysPageHandler(w http.ResponseWriter, r *http.Request) {
	physPage := r.PostFormValue("phys-page")
	// Strip all white spaces
	physPageSlice := strings.Fields(physPage)
	data, err := utils.ConvertHexStringsToBytes(physPageSlice)
	if err != nil {
		fmt.Println("Error while converting phys. page to bytes")
		http.Error(w, "Conversion to bytes failed", http.StatusBadRequest)
		return
	}
	pfn, err := strconv.ParseUint(utils.Virt2Phys("0x"+currentMemPage.Addresses[0], pid)[2:], 16, 64)
	if err != nil {
		fmt.Println("Error", err)
		http.Error(w, "Error parsing pfn in SavePhysPageHandler", http.StatusBadRequest)
		return
	}
	// Write page to memory
	utils.WritePhysPage(pfn>>12, data)
}

func CloseModalHandler(w http.ResponseWriter, r *http.Request) {
	context := make(map[string]interface{})
	tmpl := template.New("modal")
	tmpl.Parse(`<div id="modal" class="hidden"></div>`)
	tmpl.ExecuteTemplate(w, "modal", context)
}

func CloseInfoModalHandler(w http.ResponseWriter, r *http.Request) {
	context := make(map[string]interface{})
	tmpl := template.New("info-modal")
	tmpl.Parse(`<div id="info-modal" class="hidden"></div>`)
	tmpl.Execute(w, context)
}

func DumpPhysPagesHandler(w http.ResponseWriter, r *http.Request) {
	var physPages []utils.Page
	physPages = utils.GetAllPhysPages(pid)

	pageMap := make(map[string]interface{})

	// Populate the map with the data from the slice
	for idx, page := range physPages {
		key := fmt.Sprintf("page-%d", idx)

		pageMap[key] = map[string]interface{}{
			"content":     page.Content,
			"vfn":         page.Vfn,
			"translation": page.Translation,
		}
	}

	jsonBytes, err := json.Marshal(pageMap)
	if err != nil {
		http.Error(w, "Failed to encode JSON data", http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", "data.json"))
	w.Write(jsonBytes)
}

func DownloadPhysPageHandler(w http.ResponseWriter, r *http.Request) {
	strBuilder := strings.Builder{}

	// Formatting page
	for i, b := range currentMemPage.PageBytes {
		if (i % 16) == 15 {
			strBuilder.WriteString(fmt.Sprintf("%02x\n", b))
		} else {
			strBuilder.WriteString(fmt.Sprintf("%02x ", b))
		}
	}
	w.Header().Set("Content-Type", "application/text")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", "page.txt"))
	w.Write([]byte(strBuilder.String()))
}

func UploadPhysPageHandler(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("file")
	if err != nil {
		fmt.Printf("Couldn't open uploaded file %s", err)
		http.Error(w, "Couldn't open uploaded file", http.StatusBadRequest)
		return
	}
	defer file.Close()
	fileContent, err := io.ReadAll(file)
	if err != nil {
		fmt.Printf("Couldn't read uploaded file %s", err)
		http.Error(w, "Couldn't read uploaded file", http.StatusBadRequest)
		return
	}
	cleanHexString := strings.Fields(string(fileContent))
	data, err := utils.ConvertHexStringsToBytes(cleanHexString)
	if err != nil {
		fmt.Printf("Couldn't convert file data: %s", err)
		http.Error(w, "Couldn't convert file data", http.StatusBadRequest)
		return
	}
	pfn, err := strconv.ParseUint(utils.Virt2Phys("0x"+currentMemPage.Addresses[0], pid)[2:], 16, 64)
	if err != nil {
		fmt.Println("Error", err)
		http.Error(w, "Error parsing pfn in SavePhysPageHandler", http.StatusBadRequest)
		return
	}
	utils.WritePhysPage(pfn>>12, data)
}

func ShowInfoModalHandler(w http.ResponseWriter, r *http.Request) {
	templ := template.Must(template.ParseFiles("templates/info-modal.html"))
	templ.ExecuteTemplate(w, "info-modal", nil)
}

func ShowProcessMapsHandler(w http.ResponseWriter, r *http.Request) {
	file, err := os.Open(fmt.Sprintf("/proc/%d/maps", pid))
	if err != nil {
		fmt.Println("Cannot open maps file")
		http.Error(w, "Cannot open maps file", http.StatusInternalServerError)
	}
	defer file.Close()
	strBuilder := strings.Builder{}
	// Read maps file for the process
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		strBuilder.WriteString(fmt.Sprintf("%s\n", line))
	}
	context := make(map[string]interface{})
	c := strings.ReplaceAll(strBuilder.String(), "                    ", "   ")
	context["maps"] = c
	templ := template.Must(template.ParseFiles("templates/info-modal.html"))
	templ.ExecuteTemplate(w, "info-content", context)
}

func ShowBinarySectionsHandler(w http.ResponseWriter, r *http.Request) {
	context := make(map[string]interface{})
	// Program path on the system
	progPath, err := utils.GetProgPath(pid)
	if err != nil {
		http.Error(w, "Cannot get process name", http.StatusInternalServerError)
		return
	}
	// Define command (show binary sections)
	cmd := exec.Command("objdump", "-h", progPath)
	// Execute command and return output
	output, err := cmd.CombinedOutput()
	var o string
	if err != nil {
		o = fmt.Sprintf("Failed to execute \"%s\".\nCheck if objdump exists on your system!", cmd)
	} else {
		o = string(output)
	}
	context["maps"] = o
	templ := template.Must(template.ParseFiles("templates/info-modal.html"))
	templ.ExecuteTemplate(w, "info-content", context)
}

// ******** Interactive GDB Section ********

var (
	gdbCmd           *exec.Cmd
	stdinPipe        io.WriteCloser
	stdoutPipe       io.ReadCloser
	mu               sync.Mutex // Mutex for synchronizing access to strBuilder
	gdbSessionActive bool
	gdbProcess       *os.Process
	gdbOutput        []string
	ch               chan string
)

func readChannel() {
	var shouldBreak bool

	for {
		// non-blocking select needed here
		// either reads a value or breaks
		// out of the loop if no value was
		// read after one second
		select {
		case val := <-ch:
			if strings.Contains(val, "(gdb) ") {
				val = strings.ReplaceAll(val, "(gdb) ", "")
			}
			gdbOutput = append(gdbOutput, val)
		case <-time.After(time.Second):
			shouldBreak = true
		}

		// break doesn't work inside select
		if shouldBreak {
			break
		}
	}
}

// Called concurrently
func readStdout() {
	stdoutScanner := bufio.NewScanner(stdoutPipe)
	for stdoutScanner.Scan() {
		line := stdoutScanner.Text()

		mu.Lock()
		ch <- line
		mu.Unlock()
	}
}

func AttachGDBHandler(w http.ResponseWriter, r *http.Request) {
	if !gdbSessionActive {
		gdbOutput = make([]string, 0)
		gdbCmd = exec.Command("gdb", "--pid", fmt.Sprintf("%d", pid))

		// Get the pipes for stdin and stdout
		var err error
		stdinPipe, err = gdbCmd.StdinPipe()
		if err != nil {
			http.Error(w, "Error getting StdinPipe", http.StatusInternalServerError)
			return
		}

		stdoutPipe, err = gdbCmd.StdoutPipe()
		if err != nil {
			http.Error(w, "Error getting StdoutPipe", http.StatusInternalServerError)
			return
		}

		// Start the GDB process
		if err := gdbCmd.Start(); err != nil {
			http.Error(w, "Error starting GDB", http.StatusInternalServerError)
			return
		}
		// Get process instance
		gdbProcess = gdbCmd.Process

		// Channel to communicate data from go routine to main process
		ch = make(chan string)
		go readStdout()
		readChannel()
		gdbSessionActive = true
	}

	combinedOutput := strings.Join(gdbOutput, "\n")
	context := map[string]interface{}{"maps": combinedOutput, "gdb": "true"}
	templ := template.Must(template.ParseFiles("templates/info-modal.html"))
	templ.ExecuteTemplate(w, "info-content", context)
}

func SendCommandHandler(w http.ResponseWriter, r *http.Request) {
	// For gdb view in template
	gdb := "true"
	sigint := r.PostFormValue("sigint")
	// Send SIGINT to GDB
	if sigint == "true" {
		if gdbProcess != nil {
			if err := gdbProcess.Signal(syscall.SIGINT); err != nil {
				fmt.Println("Failed to send SIGINT:", err)
			}
		}
	} else {
		command := r.PostFormValue("command")
		if command == "" {
			http.Error(w, "Command is required", http.StatusBadRequest)
			return
		}

		// Ensure GDB isn't stuck before exiting
		if command == "q" || command == "quit" {
			if gdbProcess != nil {
				if err := gdbProcess.Signal(syscall.SIGINT); err != nil {
					fmt.Println("Failed to send SIGINT:", err)
				}
			}
		}

		// Send command to stdin
		if _, err := stdinPipe.Write([]byte(command + "\n")); err != nil {
			http.Error(w, "Couldn't execute command on GDB", http.StatusInternalServerError)
			fmt.Println("Couldn't execute command on GDB")
			return
		}

		// Append command and output for the template
		gdbOutput = append(gdbOutput, fmt.Sprintf("(gdb) %s", command))
		readChannel()

		if command == "q" || command == "quit" {
			gdb = ""
			gdbSessionActive = false
			close(ch)
			stdinPipe.Write([]byte("y\n"))
		}
	}

	combinedOutput := strings.Join(gdbOutput, "\n")

	context := map[string]interface{}{"maps": combinedOutput, "gdb": gdb}
	templ := template.Must(template.ParseFiles("templates/info-modal.html"))
	templ.ExecuteTemplate(w, "info-content", context)
}
