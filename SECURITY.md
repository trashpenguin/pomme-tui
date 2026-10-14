# Security reports

Do not post authentication tokens or credentials in public issues. Use the repository owner's private security reporting channel when available. If no private channel is configured, ask for one without including sensitive details.

The Apple session is stored locally under the Windows user's AppData profile. Pomme does not currently encrypt that file itself. Someone with access to the same Windows account can access it. Sign out to clear the session before sharing a machine/profile.

This unofficial early release has received targeted review and tests, not a formal independent security audit. Keep Chrome, WebView2, Windows, and the Visual C++ runtime updated through their official channels.
