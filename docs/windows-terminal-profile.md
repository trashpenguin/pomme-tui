# Pomme terminal tab icon

The apple icon beside the Pomme tab title belongs to Windows Terminal's profile,
not to the TUI header. Pomme.exe and the pomme.cmd shim select the dedicated Pomme
profile using its stable GUID. The profile points to the bundled apple PNG and the
current launcher. It does not change another profile's icon or the default profile.

The launcher writes only its own files under:
`%LOCALAPPDATA%\Microsoft\Windows Terminal\Fragments\PommeTUI`.
Unchanged files are left alone; updates use temporary files and atomic replacement.
On a changed fragment it updates the modification timestamp of known Terminal
settings files to request a reload, preserving their contents. Existing tabs keep
the profile they opened with, so close/reopen Pomme to use the new icon.

An internal child marker prevents reopening recursively. Version/doctor/help
commands stay in the caller's console and never pause on a diagnostic failure.
If Windows Terminal is unavailable, the player uses the existing console.

Validation: temporary-folder registration tests preserve sample JSONC settings,
full Go tests/vet, and an explicit real Windows Terminal 1.24 host probe. The probe
verified the Pomme profile GUID, spaced script paths/arguments, and the actual
child launcher's version response without account access or audio playback.

References:
- [Microsoft: Terminal fragment extensions](https://learn.microsoft.com/en-us/windows/terminal/json-fragment-extensions)
- [Microsoft: profile icon settings](https://learn.microsoft.com/en-us/windows/terminal/customize-settings/profile-appearance#icons)
- [Microsoft: Terminal command-line arguments](https://learn.microsoft.com/en-us/windows/terminal/command-line-arguments)
