// Command fixajfsp registers an appleJuice GUI program for ajfsp links and
// .ajl files, with the same registry entries as the appleJuice setups. It is
// meant for portable installations that no setup has registered.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"github.com/applejuicenetz/fixajfsp/internal/ajreg"
)

const (
	idPath    = 101
	idBrowse  = 102
	idApply   = 103
	idStatus  = 105
	idCurrent = 106

	wmDestroy = 0x0002
	wmClose   = 0x0010
	wmCommand = 0x0111
	wmSetFont = 0x0030
	enChange  = 0x0300
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	comdlg   = syscall.NewLazyDLL("comdlg32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	pCreateWindow = user32.NewProc("CreateWindowExW")
	pDefWindow    = user32.NewProc("DefWindowProcW")
	pSend         = user32.NewProc("SendMessageW")
	pSetText      = user32.NewProc("SetWindowTextW")
	pGetText      = user32.NewProc("GetWindowTextW")
	pMessageBox   = user32.NewProc("MessageBoxW")
)

func u16(s string) *uint16 { return syscall.StringToUTF16Ptr(s) }

func call(p *syscall.LazyProc, args ...uintptr) uintptr {
	r, _, _ := p.Call(args...)
	return r
}

func setText(h uintptr, s string) {
	t := u16(s)
	call(pSetText, h, uintptr(unsafe.Pointer(t)))
	runtime.KeepAlive(t)
}

func getText(h uintptr) string {
	buf := make([]uint16, 4096)
	call(pGetText, h, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return strings.TrimSpace(syscall.UTF16ToString(buf))
}

func messageBox(owner uintptr, text string, flags uintptr) {
	t, c := u16(text), u16("fixajfsp")
	call(pMessageBox, owner, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(c)), flags)
	runtime.KeepAlive(t)
	runtime.KeepAlive(c)
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

// browse opens the file dialog and returns the chosen .exe or "".
func browse(owner uintptr, initial string) string {
	type openFileName struct {
		StructSize      uint32
		Owner           uintptr
		Instance        uintptr
		Filter          *uint16
		CustomFilter    *uint16
		MaxCustomFilter uint32
		FilterIndex     uint32
		File            *uint16
		MaxFile         uint32
		FileTitle       *uint16
		MaxFileTitle    uint32
		InitialDir      *uint16
		Title           *uint16
		Flags           uint32
		FileOffset      uint16
		FileExtension   uint16
		DefExt          *uint16
		CustData        uintptr
		Hook            uintptr
		TemplateName    *uint16
		Reserved        uintptr
		ReservedDword   uint32
		FlagsEx         uint32
	}
	filter := syscall.StringToUTF16("Programme (*.exe)\x00*.exe\x00Alle Dateien (*.*)\x00*.*\x00")
	file := make([]uint16, 32768)
	dir := ""
	if initial != "" {
		copy(file, syscall.StringToUTF16(initial))
		dir = filepath.Dir(initial)
	}
	ofn := openFileName{
		Owner:       owner,
		Filter:      &filter[0],
		FilterIndex: 1,
		File:        &file[0],
		MaxFile:     uint32(len(file)),
		Title:       u16("appleJuice GUI wählen (z.B. GUI.exe)"),
		DefExt:      u16("exe"),
		// OFN_NOCHANGEDIR | OFN_PATHMUSTEXIST | OFN_FILEMUSTEXIST | OFN_EXPLORER
		Flags: 0x8 | 0x800 | 0x1000 | 0x80000,
	}
	if dir != "" {
		ofn.InitialDir = u16(dir)
	}
	ofn.StructSize = uint32(unsafe.Sizeof(ofn))
	if call(comdlg.NewProc("GetOpenFileNameW"), uintptr(unsafe.Pointer(&ofn))) == 0 {
		return ""
	}
	return syscall.UTF16ToString(file)
}

func report(items []ajreg.Item) string {
	var b strings.Builder
	for _, it := range items {
		name := it.Name
		if name == "" {
			name = "(Standard)"
		}
		tag := "[ OK       ]"
		switch it.State {
		case ajreg.Missing:
			tag = "[ FEHLT    ]"
		case ajreg.Different:
			tag = "[ ABWEICHT ]"
		}
		fmt.Fprintf(&b, "%s HKLM\\%s  %s\r\n", tag, it.Key, name)
		if it.State == ajreg.Different {
			fmt.Fprintf(&b, "             ist:  %s\r\n             soll: %s\r\n", it.Actual, it.Data)
		} else if it.State == ajreg.OK {
			fmt.Fprintf(&b, "             %s\r\n", it.Data)
		}
	}
	return b.String()
}

func main() {
	runtime.LockOSThread()
	store := hklmStore{}

	initial := ajreg.CurrentExe(store)
	// A registered handler is started with the clicked link as argument; only
	// an argument that is a program path preselects the file.
	if len(os.Args) > 1 && strings.HasSuffix(strings.ToLower(os.Args[1]), ".exe") {
		initial = os.Args[1]
	} else if initial == "" {
		if self, err := os.Executable(); err == nil {
			initial = ajreg.Guess(filepath.Dir(self), fileExists)
			if initial == "" {
				initial = ajreg.Guess(filepath.Dir(filepath.Dir(self)), fileExists)
			}
		}
	}

	var path, status, current uintptr
	refresh := func() {
		exe := getText(path)
		items, err := ajreg.Inspect(store, strings.Trim(exe, `"`))
		if err != nil {
			setText(status, "Registry konnte nicht gelesen werden: "+err.Error())
			return
		}
		if reg := ajreg.CurrentExe(store); reg != "" {
			note := ""
			if !fileExists(reg) {
				note = "  (Datei existiert nicht mehr)"
			}
			setText(current, "Aktuell für ajfsp:// registriert: "+reg+note)
		} else {
			setText(current, "Aktuell ist kein Programm für ajfsp:// registriert.")
		}
		setText(status, report(items))
	}

	var hwnd uintptr
	proc := syscall.NewCallback(func(h uintptr, msg uint32, wparam, lparam uintptr) uintptr {
		switch msg {
		case wmCommand:
			id, code := wparam&0xffff, (wparam>>16)&0xffff
			switch {
			case id == idPath && code == enChange:
				refresh()
			case id == idBrowse:
				if p := browse(h, getText(path)); p != "" {
					setText(path, p)
				}
			case id == idApply:
				exe, err := ajreg.ValidateExe(getText(path), fileExists)
				if err == nil {
					err = ajreg.Apply(store, exe)
				}
				if err != nil {
					messageBox(h, "Fehler: "+err.Error(), 0x10)
					break
				}
				call(shell32.NewProc("SHChangeNotify"), 0x08000000, 0, 0, 0)
				setText(path, exe)
				refresh()
				messageBox(h, "Registry-Einträge wurden gesetzt für:\n"+exe, 0x40)
			}
			return 0
		case wmClose:
			call(user32.NewProc("DestroyWindow"), h)
			return 0
		case wmDestroy:
			call(user32.NewProc("PostQuitMessage"), 0)
			return 0
		}
		return call(pDefWindow, h, uintptr(msg), wparam, lparam)
	})

	instance := call(kernel32.NewProc("GetModuleHandleW"), 0)
	type windowClass struct {
		Size        uint32
		Style       uint32
		Proc        uintptr
		ClassExtra  int32
		WindowExtra int32
		Instance    uintptr
		Icon        uintptr
		Cursor      uintptr
		Background  uintptr
		Menu        *uint16
		Name        *uint16
		SmallIcon   uintptr
	}
	className := u16("fixajfspWindow")
	// rsrc assigns ID 1 to the manifest and ID 2 to the icon group.
	icon := call(user32.NewProc("LoadImageW"), instance, 2, 1, 32, 32, 0)
	small := call(user32.NewProc("LoadImageW"), instance, 2, 1, 16, 16, 0)
	cursor := call(user32.NewProc("LoadCursorW"), 0, 32512)
	wc := windowClass{Proc: proc, Instance: instance, Icon: icon, SmallIcon: small, Cursor: cursor, Background: 6, Name: className}
	wc.Size = uint32(unsafe.Sizeof(wc))
	if call(user32.NewProc("RegisterClassExW"), uintptr(unsafe.Pointer(&wc))) == 0 {
		messageBox(0, "Fenster konnte nicht erstellt werden.", 0x10)
		os.Exit(1)
	}

	create := func(class string, text string, style, exStyle uintptr, x, y, w, h int, id uintptr) uintptr {
		c, t := u16(class), u16(text)
		r := call(pCreateWindow, exStyle, uintptr(unsafe.Pointer(c)), uintptr(unsafe.Pointer(t)), style,
			uintptr(x), uintptr(y), uintptr(w), uintptr(h), hwnd, id, instance, 0)
		runtime.KeepAlive(c)
		runtime.KeepAlive(t)
		return r
	}

	// The manifest makes the process system DPI aware: coordinates are physical
	// pixels and every size below is scaled from its 96 DPI design value.
	dpi := 96
	if r := call(user32.NewProc("GetDpiForSystem")); r != 0 {
		dpi = int(r)
	}
	px := func(v int) int { return v * dpi / 96 }
	font := func(face string, points int) uintptr {
		return call(gdi32.NewProc("CreateFontW"), uintptr(-px(points*96/72)), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 5, 0,
			uintptr(unsafe.Pointer(u16(face))))
	}

	// WS_CAPTION | WS_SYSMENU | WS_MINIMIZEBOX: fixed size, no maximize.
	const style = 0x00CA0000
	rect := struct{ Left, Top, Right, Bottom int32 }{0, 0, int32(px(784)), int32(px(562))}
	call(user32.NewProc("AdjustWindowRectEx"), uintptr(unsafe.Pointer(&rect)), style, 0, 0)
	hwnd = create("fixajfspWindow", "fixajfsp - appleJuice ajfsp:// Registrierung", style, 0, 200, 120,
		int(rect.Right-rect.Left), int(rect.Bottom-rect.Top), 0)
	if hwnd == 0 {
		messageBox(0, "Fenster konnte nicht erstellt werden.", 0x10)
		os.Exit(1)
	}
	const child = 0x50000000 // WS_CHILD | WS_VISIBLE
	const tab = 0x10000      // WS_TABSTOP
	ui := font("Segoe UI", 9)
	mono := font("Consolas", 9)
	add := func(class, text string, style uintptr, x, y, w, h int, id uintptr) uintptr {
		return create(class, text, style, 0, px(x), px(y), px(w), px(h), id)
	}

	current = add("STATIC", "", child, 12, 10, 760, 20, idCurrent)
	add("STATIC", "GUI-Programm (z.B. GUI.exe der Portable-Version):", child, 12, 36, 760, 20, 0)
	path = add("EDIT", initial, child|tab|0x800000|0x80, 12, 58, 640, 26, idPath)
	add("BUTTON", "Durchsuchen...", child|tab, 662, 57, 110, 28, idBrowse)
	add("STATIC", "Registry-Einträge (HKLM) - Ist-Zustand für das gewählte Programm:", child, 12, 94, 760, 20, 0)
	status = add("EDIT", "", child|0x800000|0x200000|0x100000|0x800|0x4|0x40|0x80, 12, 116, 760, 390, idStatus)
	add("BUTTON", "Registry-Einträge setzen", child|tab|0x1, 12, 518, 200, 32, idApply)
	add("STATIC", "Benötigt Administratorrechte. Gilt für ajfsp://, web+ajfsp:// und .ajl.", child, 230, 526, 540, 20, 0)

	call(pSend, hwnd, wmSetFont, ui, 1)
	call(user32.NewProc("EnumChildWindows"), hwnd, syscall.NewCallback(func(c, _ uintptr) uintptr {
		f := ui
		if c == status {
			f = mono
		}
		call(pSend, c, wmSetFont, f, 1)
		return 1
	}), 0)

	refresh()
	call(user32.NewProc("ShowWindow"), hwnd, 5)
	call(user32.NewProc("SetForegroundWindow"), hwnd)

	type message struct {
		Window  uintptr
		ID      uint32
		WParam  uintptr
		LParam  uintptr
		Time    uint32
		X, Y    int32
		Private uint32
	}
	var m message
	for {
		if int32(call(user32.NewProc("GetMessageW"), uintptr(unsafe.Pointer(&m)), 0, 0, 0)) <= 0 {
			break
		}
		if call(user32.NewProc("IsDialogMessageW"), hwnd, uintptr(unsafe.Pointer(&m))) != 0 {
			continue
		}
		call(user32.NewProc("TranslateMessage"), uintptr(unsafe.Pointer(&m)))
		call(user32.NewProc("DispatchMessageW"), uintptr(unsafe.Pointer(&m)))
	}
	runtime.KeepAlive(proc)
}
