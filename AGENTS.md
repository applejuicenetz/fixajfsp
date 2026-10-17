# Agent Instructions fixajfsp

@../project/AGENTS.md

Windows-GUI-Tool in Go, ersetzt das alte `fixajfsp.exe`. Registriert eine appleJuice GUI-EXE für `ajfsp://`, `web+ajfsp://` und `.ajl` in `HKLM`.

## Regeln

- Nur Standardbibliothek, keine Drittmodule. Native Win32-Controls über `syscall`, kein CGO.
- Dokumentation und Kommentare Englisch, `README.md` Deutsch und nur nutzerrelevant. Keine Emojis. Produktname appleJuice. Lizenz MIT.
- `core-src` nie referenzieren.
- Registry-Layout bleibt gleich zu den Setups (`setup`, `gui-java`: `scripts/package.py`, `gui-juicer`: `packaging/juicer.nsi`). Änderungen dort gemeinsam nachziehen.

## Layout

| Pfad | Zweck |
| --- | --- |
| `internal/ajreg` | plattformunabhängig: gewünschte Registry-Werte (`Desired`), Soll/Ist-Vergleich (`Inspect`), `Apply`, Pfadprüfung, Vorschlag der GUI-EXE (`Guess`) über das Interface `Store` |
| `cmd/fixajfsp/store_windows.go` | `Store` auf `HKEY_LOCAL_MACHINE`, immer 64-Bit-Ansicht (`KEY_WOW64_64KEY`) |
| `cmd/fixajfsp/main_windows.go` | Win32-Fenster: Anzeige Ist-Zustand, Pfadfeld, Datei-Dialog, Setzen-Button |
| `cmd/fixajfsp/fixajfsp.manifest` | verlangt Administratorrechte, Common Controls 6 |

## Build und Tests

Go 1.27 ist nicht auf dem Host installiert. `build.sh` prüft gofmt, vet, Tests und baut in Docker `dist/fixajfsp-windows-<amd64|arm64>.exe` (`-H windowsgui`). Die `.syso`-Ressourcen (Icon, Manifest) werden dabei mit `rsrc` erzeugt und nicht versioniert.

Getestet wird nur `internal/ajreg` automatisiert. Das Fenster und der Registry-Zugriff sind gebaut, aber nicht automatisiert geprüft; manuell unter Windows x64 und ARM64 testen (Start als Admin, Auswahl, Setzen, `ajfsp://`-Link im Browser öffnen).

## Release

`.github/workflows/build.yml` baut bei Push und Pull Request. Ein gepushter Tag (`1.0.0`) veröffentlicht beide EXE als Release.
