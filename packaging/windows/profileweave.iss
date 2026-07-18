#ifndef AppVersion
  #error AppVersion must be supplied by the build script
#endif
#ifndef PayloadDir
  #error PayloadDir must be supplied by the build script
#endif
#ifndef OutputDir
  #error OutputDir must be supplied by the build script
#endif
#ifndef OutputBaseFilename
  #error OutputBaseFilename must be supplied by the build script
#endif

#ifdef TargetAMD64
  #define ArchitectureAllowed "x64compatible and not arm64"
  #define ArchitectureInstallMode "x64compatible"
#elif defined(TargetARM64)
  #define ArchitectureAllowed "arm64"
  #define ArchitectureInstallMode "arm64"
#else
  #error Define exactly one of TargetAMD64 or TargetARM64
#endif

#define ProductName "ProfileWeave"
#define ProductExe "profileweave.exe"
#define ProductAppId "{E6784F49-0EE8-4D07-A8D9-E9A64B1D53B8}"
#define ProductURL "https://github.com/miahwk/profileweave"

[Setup]
AppId={{#ProductAppId}
AppName={#ProductName}
AppVersion={#AppVersion}
AppPublisher=ProfileWeave contributors
AppPublisherURL={#ProductURL}
AppSupportURL={#ProductURL}/issues
AppUpdatesURL={#ProductURL}/releases
DefaultDirName={localappdata}\Programs\{#ProductName}
DefaultGroupName={#ProductName}
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
MinVersion=10.0.17763
ArchitecturesAllowed={#ArchitectureAllowed}
ArchitecturesInstallIn64BitMode={#ArchitectureInstallMode}
OutputDir={#OutputDir}
OutputBaseFilename={#OutputBaseFilename}
Compression=lzma2/max
SolidCompression=yes
WizardStyle=modern
LicenseFile={#PayloadDir}\LICENSE
UninstallDisplayName={#ProductName}
UninstallDisplayIcon={app}\{#ProductExe}
CloseApplications=yes
RestartApplications=no
RestartIfNeededByRun=no
ChangesAssociations=no
ChangesEnvironment=no
AllowNoIcons=no

[Tasks]
Name: "desktopicon"; Description: "Create a &desktop shortcut"; GroupDescription: "Additional shortcuts:"; Flags: unchecked

[Files]
Source: "{#PayloadDir}\{#ProductExe}"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#PayloadDir}\frontend\dist\*"; DestDir: "{app}\frontend\dist"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "{#PayloadDir}\README.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#PayloadDir}\LICENSE"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#PayloadDir}\NOTICE"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#PayloadDir}\THIRD_PARTY_NOTICES.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#PayloadDir}\CHANGELOG.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#PayloadDir}\SECURITY.md"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\{#ProductName}"; Filename: "{app}\{#ProductExe}"; Parameters: "--open --log-file ""{localappdata}\ProfileWeave\logs\profileweave.log"""; WorkingDir: "{app}"
Name: "{group}\Exit {#ProductName}"; Filename: "{app}\{#ProductExe}"; Parameters: "--shutdown --log-file ""{localappdata}\ProfileWeave\logs\profileweave.log"""; WorkingDir: "{app}"
Name: "{autodesktop}\{#ProductName}"; Filename: "{app}\{#ProductExe}"; Parameters: "--open --log-file ""{localappdata}\ProfileWeave\logs\profileweave.log"""; WorkingDir: "{app}"; Tasks: desktopicon

[Run]
Filename: "{app}\{#ProductExe}"; Parameters: "--open --log-file ""{localappdata}\ProfileWeave\logs\profileweave.log"""; WorkingDir: "{app}"; Description: "Launch {#ProductName}"; Flags: nowait postinstall skipifsilent

[Code]
function ShutdownInstance(): Boolean;
var
  ResultCode: Integer;
  ExecutablePath: String;
begin
  Result := True;
  ExecutablePath := ExpandConstant('{app}\{#ProductExe}');
  if not FileExists(ExecutablePath) then
    exit;

  Result := Exec(
    ExecutablePath,
    '--shutdown',
    ExpandConstant('{app}'),
    SW_HIDE,
    ewWaitUntilTerminated,
    ResultCode
  ) and (ResultCode = 0);
end;

function PrepareToInstall(var NeedsRestart: Boolean): String;
begin
  Result := '';
  if not ShutdownInstance() then
    Result := 'ProfileWeave could not be stopped safely. Close it and retry the installation.';
end;

function InitializeUninstall(): Boolean;
begin
  Result := ShutdownInstance();
  if not Result then
    SuppressibleMsgBox(
      'ProfileWeave could not be stopped safely. Close it and retry the uninstall. Your profile data has not been changed.',
      mbError,
      MB_OK,
      IDOK
    );
end;
