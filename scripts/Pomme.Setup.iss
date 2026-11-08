#ifndef AppVersion
  #error AppVersion must be provided by Build-Setup.ps1
#endif
#ifndef ReleaseDir
  #error ReleaseDir must be provided by Build-Setup.ps1
#endif
#ifndef OutputDir
  #error OutputDir must be provided by Build-Setup.ps1
#endif

[Setup]
AppId={{6A02AC7C-757E-4540-A60C-267CF9702C2D}
AppName=Pomme TUI
AppVersion={#AppVersion}
AppPublisher=Pomme TUI contributors
AppPublisherURL=https://github.com/trashpenguin/pomme-tui
AppSupportURL=https://github.com/trashpenguin/pomme-tui/issues
AppUpdatesURL=https://github.com/trashpenguin/pomme-tui/releases
DefaultDirName={localappdata}\Programs\PommeTUI
DefaultGroupName=Pomme TUI
PrivilegesRequired=lowest
SetupArchitecture=x64
ArchitecturesAllowed=x64os
ArchitecturesInstallIn64BitMode=x64os
MinVersion=10.0
DisableProgramGroupPage=yes
AllowNoIcons=yes
LicenseFile=..\LICENSE
SetupIconFile=..\assets\pomme.ico
UninstallDisplayIcon={app}\Pomme.exe
OutputDir={#OutputDir}
OutputBaseFilename=pomme-tui-{#AppVersion}-setup-x64
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
CloseApplications=yes
RestartApplications=no
SetupMutex=PommeTUISetup

[Tasks]
Name: desktopicon; Description: "Create a desktop shortcut"; Flags: unchecked

[Files]
Source: "{#ReleaseDir}\*"; DestDir: "{app}"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "Pomme.Helpers.ps1"; DestName: "Setup.Pomme.Helpers.ps1"; Flags: dontcopy
Source: "Setup.Helpers.ps1"; Flags: dontcopy
Source: "Setup-Dependencies.ps1"; Flags: dontcopy

[Icons]
Name: "{group}\Pomme TUI"; Filename: "{app}\Pomme.exe"; WorkingDir: "{app}"
Name: "{autodesktop}\Pomme TUI"; Filename: "{app}\Pomme.exe"; WorkingDir: "{app}"; Tasks: desktopicon

[Run]
Filename: "{app}\Pomme.exe"; Description: "Open Pomme and sign in to Apple Music"; Flags: nowait postinstall skipifsilent; Check: CanLaunch

[Code]
var
  PrerequisitesPage: TOutputMsgWizardPage;
  RestartRequired: Boolean;
  DetectionFailed: Boolean;

function DependencyCommand(Install: Boolean): String;
begin
  Result := '-NoProfile -NonInteractive -ExecutionPolicy Bypass -File "' +
    ExpandConstant('{tmp}\Setup-Dependencies.ps1') + '" -ResultPath "' +
    ExpandConstant('{tmp}\pomme-prerequisites.txt') + '"';
  if Install then Result := Result + ' -Install';
end;

function DependencyMessage: String;
var
  Text: AnsiString;
begin
  Result := 'Unable to check required components. Close setup and try again.';
  if LoadStringFromFile(ExpandConstant('{tmp}\pomme-prerequisites.txt'), Text) then
    Result := String(Text);
end;

procedure InitializeWizard;
var
  Code: Integer;
begin
  ExtractTemporaryFile('Setup.Pomme.Helpers.ps1');
  ExtractTemporaryFile('Setup.Helpers.ps1');
  ExtractTemporaryFile('Setup-Dependencies.ps1');
  PrerequisitesPage := CreateOutputMsgPage(wpLicense, 'Ready for Apple Music',
    'Setup takes care of the required components.', '');
  DetectionFailed := not Exec(ExpandConstant('{sys}\WindowsPowerShell\v1.0\powershell.exe'),
    DependencyCommand(False), '', SW_HIDE, ewWaitUntilTerminated, Code);
  if not DetectionFailed then DetectionFailed := (Code <> 0) and (Code <> 10);
  PrerequisitesPage.MsgLabel.Caption := DependencyMessage + #13#10#13#10 +
    'If needed, setup downloads Google Chrome (with Widevine), Microsoft Visual C++ x64 Runtime and Microsoft WebView2 directly from their publishers. Windows may ask for administrator permission.' + #13#10#13#10 +
    'Continuing authorizes these prerequisite installations under their publishers'' terms. Chrome is installed only if an installed Widevine module cannot be found.' + #13#10#13#10 +
    'An active Apple Music subscription and your own Apple sign-in are required. An internet connection is required. Setup never restarts Windows automatically.';
end;

function NextButtonClick(CurPageID: Integer): Boolean;
begin
  Result := True;
  if (CurPageID = PrerequisitesPage.ID) and DetectionFailed and not WizardSilent then begin
    SuppressibleMsgBox(DependencyMessage, mbError, MB_OK, IDOK);
    Result := False;
  end;
end;

function PrepareToInstall(var NeedsRestart: Boolean): String;
var
  Code: Integer;
begin
  Result := '';
  WizardForm.PreparingLabel.Caption := 'Installing missing components. Please allow any Windows permission prompts. Downloads may take several minutes.';
  if not Exec(ExpandConstant('{sys}\WindowsPowerShell\v1.0\powershell.exe'),
    DependencyCommand(True), '', SW_HIDE, ewWaitUntilTerminated, Code) then begin
    Result := 'Could not start prerequisite setup: ' + SysErrorMessage(Code);
  end else if Code = 3010 then begin
    RestartRequired := True;
  end else if Code <> 0 then begin
    Result := DependencyMessage;
  end;
end;

function NeedRestart: Boolean;
begin
  // Inno may reboot silently if this returns True. The user must restart manually.
  Result := False;
end;

function GetCustomSetupExitCode: Integer;
begin
  if RestartRequired then Result := 3010 else Result := 0;
end;

procedure CurPageChanged(CurPageID: Integer);
begin
  if (CurPageID = wpFinished) and RestartRequired then
    WizardForm.FinishedLabel.Caption := 'Pomme is installed. Restart Windows before opening it to finish installing required components. Setup will not restart Windows for you.';
end;

function CanLaunch: Boolean;
begin
  Result := not RestartRequired;
end;
