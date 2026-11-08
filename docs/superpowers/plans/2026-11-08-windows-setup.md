# Windows setup implementation plan

Goal: one Windows setup download that installs Pomme, obtains missing official prerequisites, adds shortcuts and opens sign-in through the player. Keep the portable ZIP.

Architecture: Inno Setup 7.1.1 packages the existing release directory. A PowerShell prerequisite helper shares launcher detection, downloads official Google/Microsoft installers only when needed, checks trusted publisher signatures, installs and rechecks dependencies. Pomme does not redistribute Widevine separately. Subscription/sign-in and playback feature limits remain unchanged.

1. Add failing PowerShell tests for prerequisite skip, rejected signatures, failed installers, detection after installation and restart requirements. Implement the helper, sharing existing dependency detection.
2. Add the native setup definition and a reproducible compiler bootstrap with pinned hash/publisher verification. Integrate the setup executable into release checksums and CI artifacts.
3. Run prerequisite/launcher and full release checks. Test the actual setup in an isolated installation without replacing the normal profile, icons or pre-existing uninstall registration. Inspect the official Chrome MSI for bundled x64 Widevine without installing it on the development PC.
4. Update the version, README, changelog, release notes and verification with observed results and remaining clean-machine limitations. Build matching Windows/setup/source artifacts, verify the exact commit in hosted CI, and publish as a prerelease with verified downloads.

Limits: no removal/reinstallation of working host prerequisites for testing; no new stable-release claim without clean-machine and remaining hardware/soak checks. No automatic reboot. Uninstall preserves the Pomme profile and shared Google/Microsoft runtimes.
