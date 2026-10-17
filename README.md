# fixajfsp

![](https://img.shields.io/github/license/applejuicenetz/fixajfsp)
![](https://img.shields.io/github/v/release/applejuicenetz/fixajfsp)
![](https://img.shields.io/github/downloads/applejuicenetz/fixajfsp/total)

Kleines Windows-Programm, das `ajfsp://`-Links und `.ajl`-Dateien mit einer appleJuice GUI verknüpft.
Gedacht für die [Portable-Version](https://github.com/applejuicenetz/portable) und alle Installationen, die kein Setup durchlaufen haben und deshalb keine Registry-Einträge besitzen.

## Benutzung

1. `fixajfsp-windows-amd64.exe` (oder `-arm64.exe`) starten. Windows fragt nach Administratorrechten.
2. Das Fenster zeigt den Ist-Zustand der Registry. Liegt `fixajfsp.exe` im Ordner der Portable-Version, wird die GUI automatisch vorgeschlagen.
3. Mit **Durchsuchen...** die GUI-Datei wählen, bei Portable `GUI\GUI.exe`. Die Liste zeigt sofort, welche Einträge `FEHLT`, `ABWEICHT` oder `OK` sind.
4. **Registry-Einträge setzen** schreibt alle Einträge.

Die Einträge entsprechen denen der appleJuice Setups (unter `HKLM`, 64-Bit-Ansicht):

- Protokolle `ajfsp` und `web+ajfsp` mit Symbol und Öffnen-Befehl `"<GUI.exe>" "%1"`
- Klassen `appleJuiceNETZ.URI.Portable` und `appleJuiceNETZ.EXT.Portable`
- `Capabilities` mit URL- und Dateizuordnung (`.ajl`) und Eintrag unter `RegisteredApplications`

Wird die Portable-Version verschoben, `fixajfsp` erneut starten und die neue GUI wählen.

Optional kann der Pfad als Argument übergeben werden: `fixajfsp.exe "D:\appleJuice\GUI\GUI.exe"`.

## Lizenz

MIT, siehe `LICENSE`.
