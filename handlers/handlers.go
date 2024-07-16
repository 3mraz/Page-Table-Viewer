package handlers

import (
	"cgo_test/utils"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	// "unsafe"
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
}

func getEntries(lvl uint64, present string, tmplName string, numEntries uint16) (map[uint16]utils.PTEntry, error) {
	if present != "" {
		return cstate[tmplName], nil
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
			return nil, errors.New("Couln't get entries")
		}
		vAddrs := make(map[uint16]utils.PTEntry)
		for index, entry := range entries {
			for i := uint16(0); i < numEntries; i++ {
				vfn := _calc_vfn(lvl, int64(i))
				if vfn == entry.Vfn {
					vAddrs[i] = entry
					if index != 0 {
						break
					}
				} else if index == 0 {
					e := utils.PTEntry{Vfn: vfn, Color: "red", Pfn: "0x0"}
					vAddrs[i] = e
				}
			}
		}
		cstate[tmplName] = vAddrs
		return vAddrs, nil
	}
}

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

func setup_temp(idx int64, lvl uint64, present string) (map[uint16]utils.PTEntry, string, string, string, bool, error) {
	var tmplName string
	var nxtLvlName string
	var tmplPath string
	var numEntries uint16 = 512
	var only_present bool = false
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
		only_present = !only_present_pml4 && only_present
		only_present_pml4 = only_present
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
		err = errors.New("Couldn't setup template")
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
	}

	pidHTML := fmt.Sprintf("<label id='pid' for='virt'><b>Process Id:</b> %d</label>", pid)

	tmpl, _ := template.New("pidHTML").Parse(pidHTML)
	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, "cannot execute template", http.StatusInternalServerError)
	}
}

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
	pml4Enries, pml4TmplName, pml4NxtLvlName, pml4TmplPath, _, pml4_err := setup_temp(0, 0, "")
	pdptEnries, pdptTmplName, pdptNxtLvlName, pdptTmplPath, _, pdpt_err := setup_temp(pml4Idx, 1, "")
	pdEnries, pdTmplName, pdNxtLvlName, pdTmplPath, _, pd_err := setup_temp(pdptIdx, 2, "")
	pteEnries, pteTmplName, pteNxtLvlName, pteTmplPath, _, pte_err := setup_temp(pdIdx, 3, "")
	if pml4_err != nil || pdpt_err != nil || pd_err != nil || pte_err != nil {
		fmt.Println("Error while getting entries")
		return
	}
	e := pml4Enries[uint16(pml4Idx)]
	e.Color = "blue"
	pml4Enries[uint16(pml4Idx)] = e
	e = pdptEnries[uint16(pdptIdx)]
	e.Color = "blue"
	pdptEnries[uint16(pdptIdx)] = e
	e = pdEnries[uint16(pdIdx)]
	e.Color = "blue"
	pdEnries[uint16(pdIdx)] = e
	e = pteEnries[uint16(pteIdx)]
	e.Color = "blue"
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
		e = utils.PTEntry{Vfn: entryVfn, Pfn: "0x0", Color: "red"}
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
	warnings = make([]string, 0)
	p, _ := strconv.ParseUint(r.PostFormValue("p"), 10, 64)
	if p != 1 {
		warnings = append(warnings, "Cannot edit a non-present entry.")
		warnings = append(warnings, "You may want to set the entry to present.")
		return
	}
	pfn := r.PostFormValue("pfn")
	if !utils.ValidPhys(pfn) {
		warnings = append(warnings, "The PFN you entered is not valid.")
		return
	}
	nx, _ := strconv.ParseUint(r.PostFormValue("nx"), 10, 64)
	pat, _ := strconv.ParseUint(r.PostFormValue("pat"), 10, 64)
	patl, _ := strconv.ParseUint(r.PostFormValue("patl"), 10, 64)
	s1, _ := strconv.ParseUint(r.PostFormValue("s1"), 10, 64)
	s2, _ := strconv.ParseUint(r.PostFormValue("s2"), 10, 64)
	s3, _ := strconv.ParseUint(r.PostFormValue("s3"), 10, 64)
	u, _ := strconv.ParseUint(r.PostFormValue("u"), 10, 64)
	dc, _ := strconv.ParseUint(r.PostFormValue("dc"), 10, 64)
	d, _ := strconv.ParseUint(r.PostFormValue("d"), 10, 64)
	a, _ := strconv.ParseUint(r.PostFormValue("a"), 10, 64)
	w, _ := strconv.ParseUint(r.PostFormValue("w"), 10, 64)
	wt, _ := strconv.ParseUint(r.PostFormValue("wt"), 10, 64)
	g, _ := strconv.ParseUint(r.PostFormValue("g"), 10, 64)
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

	cstate[tableName][uint16(idx)] = e
}
