package ajreg

import "testing"

type memStore map[[2]string]string

func (m memStore) Get(k, n string) (string, bool, error) {
	v, ok := m[[2]string{k, n}]
	return v, ok, nil
}

func (m memStore) Set(k, n, d string) error {
	m[[2]string{k, n}] = d
	return nil
}

const exe = `C:\appleJuice\GUI\GUI.exe`

func count(items []Item, st State) (n int) {
	for _, i := range items {
		if i.State == st {
			n++
		}
	}
	return
}

func TestApplyThenInspect(t *testing.T) {
	s := memStore{}
	items, err := Inspect(s, exe)
	if err != nil {
		t.Fatal(err)
	}
	if count(items, Missing) != len(items) {
		t.Fatal("empty store must report everything missing")
	}
	if err := Apply(s, exe); err != nil {
		t.Fatal(err)
	}
	items, _ = Inspect(s, exe)
	if count(items, OK) != len(items) {
		t.Fatalf("after apply all must be OK: %+v", items)
	}
	if got := CurrentExe(s); got != exe {
		t.Fatalf("CurrentExe = %q", got)
	}
	if got := s[[2]string{`SOFTWARE\Classes\ajfsp\shell\open\command`, ""}]; got != `"C:\appleJuice\GUI\GUI.exe" "%1"` {
		t.Fatalf("command = %q", got)
	}
}

func TestInspectDifferent(t *testing.T) {
	s := memStore{}
	_ = Apply(s, exe)
	items, _ := Inspect(s, `D:\other\GUI.exe`)
	if count(items, Different) == 0 {
		t.Fatal("other exe must differ")
	}
	items, _ = Inspect(s, "")
	if count(items, OK) != len(items) {
		t.Fatal("without exe, present values count as OK")
	}
}

func TestExeFromCommand(t *testing.T) {
	cases := map[string]string{
		`"C:\a b\GUI.exe" "%1"`: `C:\a b\GUI.exe`,
		`C:\x\GUI.EXE %1`:       `C:\x\GUI.EXE`,
		`"broken`:               "",
		"":                      "",
	}
	for in, want := range cases {
		if got := ExeFromCommand(in); got != want {
			t.Errorf("%q: got %q want %q", in, got, want)
		}
	}
}

func TestValidateExe(t *testing.T) {
	yes := func(string) bool { return true }
	no := func(string) bool { return false }
	if _, err := ValidateExe("", yes); err == nil {
		t.Error("empty")
	}
	if _, err := ValidateExe(`C:\a\b.txt`, yes); err == nil {
		t.Error("extension")
	}
	if _, err := ValidateExe(`C:\a\b.exe`, no); err == nil {
		t.Error("missing file")
	}
	if got, err := ValidateExe(`  "C:\a\b.EXE" `, yes); err != nil || got == "" {
		t.Errorf("quoted: %q %v", got, err)
	}
}

func TestGuess(t *testing.T) {
	has := func(want string) func(string) bool {
		return func(p string) bool { return p == want }
	}
	if got := Guess(`D:\aj\`, has(`D:\aj\GUI\GUI.exe`)); got != `D:\aj\GUI\GUI.exe` {
		t.Errorf("got %q", got)
	}
	if got := Guess(`D:\aj`, has(`D:\aj\AJCoreGUI.exe`)); got != `D:\aj\AJCoreGUI.exe` {
		t.Errorf("got %q", got)
	}
	if got := Guess(`D:\aj`, func(string) bool { return false }); got != "" {
		t.Errorf("got %q", got)
	}
}
