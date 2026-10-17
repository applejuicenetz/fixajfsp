// Package ajreg describes the registry entries that make Windows open
// ajfsp links and .ajl files with an appleJuice GUI, and checks or writes them
// through a Store. The layout follows the appleJuice Windows setups.
package ajreg

import (
	"errors"
	"fmt"
	"strings"
)

const (
	// URIClass and ExtClass are the unique ProgIDs of the portable GUI.
	URIClass = "appleJuiceNETZ.URI.Portable"
	ExtClass = "appleJuiceNETZ.EXT.Portable"

	capabilitiesKey = `SOFTWARE\appleJuiceNETZ\Portable\Capabilities`
	registeredKey   = `SOFTWARE\RegisteredApplications`
	registeredName  = "Portable"
	classesKey      = `SOFTWARE\Classes\`
)

// Value is one registry value below HKLM. An empty Name is the default value.
type Value struct {
	Key  string
	Name string
	Data string
}

// State compares an expected value with the registry.
type State int

const (
	Missing State = iota
	Different
	OK
)

// Item is an expected value with what the registry currently holds.
type Item struct {
	Value
	Actual string
	State  State
}

// Store reads and writes string values below HKLM.
type Store interface {
	Get(key, name string) (data string, found bool, err error)
	Set(key, name, data string) error
}

// Command returns the shell command that opens a link or file with exe.
func Command(exe string) string { return `"` + exe + `" "%1"` }

func icon(exe string) string { return `"` + exe + `",0` }

// Desired lists all values that must exist for exe, in write order.
func Desired(exe string) []Value {
	var v []Value
	protocol := func(class, title string) {
		k := classesKey + class
		v = append(v,
			Value{k, "", title},
			Value{k, "URL Protocol", ""},
			Value{k + `\DefaultIcon`, "", icon(exe)},
			Value{k + `\shell\open\command`, "", Command(exe)},
		)
	}
	protocol("ajfsp", "URL:appleJuice File Sharing Protocol")
	protocol("web+ajfsp", "URL:appleJuice File Sharing Protocol")
	protocol(URIClass, "URL:ajfsp")

	k := classesKey + ExtClass
	v = append(v,
		Value{k, "", "appleJuice Linkliste"},
		Value{k + `\DefaultIcon`, "", icon(exe)},
		Value{k + `\shell\open\command`, "", Command(exe)},
		Value{capabilitiesKey, "ApplicationName", "Portable"},
		Value{capabilitiesKey, "ApplicationIcon", icon(exe)},
		Value{capabilitiesKey, "ApplicationDescription", "appleJuice GUI"},
		Value{capabilitiesKey + `\URLAssociations`, "ajfsp", URIClass},
		Value{capabilitiesKey + `\URLAssociations`, "web+ajfsp", URIClass},
		Value{capabilitiesKey + `\FileAssociations`, ".ajl", ExtClass},
		Value{registeredKey, registeredName, `Software\appleJuiceNETZ\Portable\Capabilities`},
	)
	return v
}

// Inspect compares every expected value for exe with the registry. An empty
// exe only reports which values exist.
func Inspect(s Store, exe string) ([]Item, error) {
	want := Desired(exe)
	items := make([]Item, 0, len(want))
	for _, w := range want {
		actual, found, err := s.Get(w.Key, w.Name)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", w.Key, err)
		}
		st := OK
		switch {
		case !found:
			st = Missing
		case exe == "" || !strings.EqualFold(actual, w.Data):
			st = Different
			if exe == "" {
				st = OK
			}
		}
		items = append(items, Item{Value: w, Actual: actual, State: st})
	}
	return items, nil
}

// CurrentExe returns the program registered for ajfsp links, or "".
func CurrentExe(s Store) string {
	cmd, found, err := s.Get(classesKey+`ajfsp\shell\open\command`, "")
	if err != nil || !found {
		return ""
	}
	return ExeFromCommand(cmd)
}

// ExeFromCommand extracts the program path from a shell command line.
func ExeFromCommand(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if strings.HasPrefix(cmd, `"`) {
		if end := strings.Index(cmd[1:], `"`); end >= 0 {
			return cmd[1 : 1+end]
		}
		return ""
	}
	if i := strings.Index(strings.ToLower(cmd), ".exe"); i >= 0 {
		return cmd[:i+4]
	}
	return ""
}

// ValidateExe checks that exe is an absolute path of a .exe file and returns it
// cleaned. exists reports whether the file is present.
func ValidateExe(exe string, exists func(string) bool) (string, error) {
	exe = strings.TrimSpace(strings.Trim(strings.TrimSpace(exe), `"`))
	if exe == "" {
		return "", errors.New("keine Datei gewählt")
	}
	if !strings.HasSuffix(strings.ToLower(exe), ".exe") {
		return "", errors.New("die Datei muss eine .exe sein")
	}
	if !isAbs(exe) {
		return "", errors.New("der Pfad muss absolut sein")
	}
	if !exists(exe) {
		return "", errors.New("die Datei existiert nicht")
	}
	return exe, nil
}

// Apply writes all values for exe.
func Apply(s Store, exe string) error {
	if exe == "" {
		return errors.New("exe is empty")
	}
	for _, v := range Desired(exe) {
		if err := s.Set(v.Key, v.Name, v.Data); err != nil {
			return fmt.Errorf("write %s: %w", v.Key, err)
		}
	}
	return nil
}

// isAbs reports whether p is a Windows drive path (C:\x) or UNC path (\\host\x).
func isAbs(p string) bool {
	if strings.HasPrefix(p, `\\`) {
		return len(p) > 2
	}
	if len(p) < 3 || p[1] != ':' || (p[2] != '\\' && p[2] != '/') {
		return false
	}
	c := p[0] | 0x20
	return c >= 'a' && c <= 'z'
}

// Guess returns the most likely GUI program in dir (the folder of fixajfsp),
// or "" when none of the usual portable layouts is present.
func Guess(dir string, exists func(string) bool) string {
	if dir == "" {
		return ""
	}
	dir = strings.TrimRight(dir, `\/`)
	for _, name := range []string{`AJCoreGUI.exe`, `GUI\GUI.exe`, `GUI.exe`} {
		if p := dir + `\` + name; exists(p) {
			return p
		}
	}
	return ""
}
