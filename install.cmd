@echo off
setlocal enabledelayedexpansion
rem Install aos on Windows, installing Go first if it is missing.
rem
rem   curl -fsSL https://raw.githubusercontent.com/muthuishere/agentic-os/main/install.cmd -o install.cmd
rem   install.cmd
rem
rem A .cmd rather than a .ps1 on purpose: PowerShell's execution policy blocks
rem unsigned script FILES by default, which stops most people before they start.
rem It does not block -Command, so the fiddly parts below still use PowerShell.
rem
rem Everything installs under your user profile. Nothing here needs administrator.

set "MODULE=github.com/muthuishere/agentic-os/cmd/aos@latest"
set "GO_INSTALL_DIR=%LOCALAPPDATA%\agentic-os\go"

echo.
echo ==^> Looking for Go
where go >nul 2>&1
if %ERRORLEVEL% equ 0 (
    for /f "tokens=*" %%v in ('go env GOVERSION 2^>nul') do set "GOVER=%%v"
    echo     found !GOVER!
    goto :build
)

echo     not found, installing it
call :install_go
if %ERRORLEVEL% neq 0 exit /b 1
set "PATH=%GO_INSTALL_DIR%\go\bin;%PATH%"

:build
echo.
echo ==^> Building aos
go install %MODULE%
if %ERRORLEVEL% neq 0 (
    echo install: build failed 1>&2
    exit /b 1
)

for /f "tokens=*" %%g in ('go env GOBIN 2^>nul') do set "GOBIN=%%g"
if "%GOBIN%"=="" for /f "tokens=*" %%g in ('go env GOPATH 2^>nul') do set "GOBIN=%%g\bin"
set "AOS=%GOBIN%\aos.exe"
if not exist "%AOS%" (
    echo install: expected a binary at %AOS% 1>&2
    exit /b 1
)
echo     %AOS%

echo.
echo ==^> Installing the agent skill
"%AOS%" install --skills

echo.
echo ==^> Checking this machine
"%AOS%" doctor

where aos >nul 2>&1
if %ERRORLEVEL% equ 0 (
    echo.
    echo Ready. Try: aos --help
) else (
    echo.
    echo Ready, but %GOBIN% is not on your PATH yet.
    echo Add it for future sessions with:
    echo.
    echo     setx PATH "%GOBIN%;%%PATH%%"
    echo.
    echo Or run it directly: "%AOS%" --help
)
exit /b 0

:install_go
rem Architecture: Windows reports ARM64 on Snapdragon and Parallels-on-Apple-silicon.
set "GOARCH=amd64"
if /i "%PROCESSOR_ARCHITECTURE%"=="ARM64" set "GOARCH=arm64"

for /f "tokens=*" %%v in ('curl -fsSL "https://go.dev/VERSION?m=text" 2^>nul ^| findstr /b "go"') do (
    set "GOVERSION=%%v"
    goto :got_version
)
:got_version
if "%GOVERSION%"=="" (
    echo install: could not determine the current Go version 1>&2
    exit /b 1
)
set "ARCHIVE=%GOVERSION%.windows-%GOARCH%.zip"
echo     %GOVERSION% for windows/%GOARCH%

rem Download, verify against the checksum in the official JSON index, and expand.
rem The <archive>.sha256 URL is not usable: it answers 200 with an HTML page, so
rem comparing against it fails every time.
powershell -NoProfile -Command ^
  "$ErrorActionPreference='Stop';" ^
  "$archive='%ARCHIVE%'; $dest='%GO_INSTALL_DIR%';" ^
  "$tmp=Join-Path $env:TEMP $archive;" ^
  "Invoke-WebRequest -UseBasicParsing -Uri ('https://go.dev/dl/'+$archive) -OutFile $tmp;" ^
  "$want=((Invoke-WebRequest -UseBasicParsing -Uri 'https://go.dev/dl/?mode=json').Content | ConvertFrom-Json).files | Where-Object { $_.filename -eq $archive } | Select-Object -First 1 -ExpandProperty sha256;" ^
  "$got=(Get-FileHash -Algorithm SHA256 $tmp).Hash;" ^
  "if ($want -and $got -ne $want) { throw ('checksum mismatch for '+$archive) }" ^
  "if ($want) { Write-Host '    checksum verified' } else { Write-Host '    WARNING: could not verify the checksum' }" ^
  "if (Test-Path $dest) { Remove-Item -Recurse -Force $dest }" ^
  "New-Item -ItemType Directory -Force -Path $dest | Out-Null;" ^
  "Expand-Archive -Path $tmp -DestinationPath $dest -Force;" ^
  "Remove-Item $tmp -Force;" ^
  "Write-Host ('    installed to '+$dest)"
if %ERRORLEVEL% neq 0 (
    echo install: could not install Go 1>&2
    exit /b 1
)
exit /b 0
