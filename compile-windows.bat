@echo off
REM Windows batch wrapper for compile-windows.ps1 and prepare-deps.ps1
REM This wrapper ensures proper execution when invoked via SSH from Unix systems
REM It fixes line endings (LF to CRLF) and sets proper PowerShell execution context
REM
REM Usage: compile-windows.bat [-Windows] [-Package] [-All] etc.
REM
REM Why this wrapper exists:
REM 1. PowerShell scripts synced from Unix have LF line endings, Windows needs CRLF
REM 2. SSH sessions have different execution policy than interactive sessions
REM 3. Environment variables may differ between RDP and SSH sessions
REM
REM Solution: This wrapper fixes line endings and bypasses execution policy

cd /d "%~dp0"

REM Function to fix line endings in PowerShell scripts
REM This is critical when files are synced from Unix systems (LF) to Windows (CRLF)
REM Converts all line endings to CRLF for proper PowerShell parsing
powershell.exe -NoProfile -ExecutionPolicy Bypass -Command ^
"function Fix-LineEndings { param([string]$file); if(Test-Path $file) { $content = [System.IO.File]::ReadAllText($file, [System.Text.Encoding]::UTF8); $content = $content -replace \"`r?`n\", \"`r`n\"; [System.IO.File]::WriteAllText($file, $content, [System.Text.Encoding]::UTF8) } }; ^
Fix-LineEndings '%~dp0compile-windows.ps1'; ^
Fix-LineEndings '%~dp0prepare-deps.ps1'; ^
Fix-LineEndings '%~dp0check-windows-env.ps1'"

REM Optional: Run environment check first (uncomment to enable)
REM powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0check-windows-env.ps1"
REM if %ERRORLEVEL% NEQ 0 (
REM     echo Environment check failed. Continuing anyway...
REM )

REM Set execution policy bypass for this session and execute PowerShell script
REM -NoProfile: Skip loading user profile (faster, more consistent, avoids profile errors)
REM -ExecutionPolicy Bypass: Override execution policy restrictions
REM -File: Execute script file directly (better than -Command for complex scripts)
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0compile-windows.ps1" %*

REM Capture exit code from PowerShell
set EXITCODE=%ERRORLEVEL%

REM Exit with the same code as PowerShell
exit /b %EXITCODE%

REM "Now this is not even the end. It is not even the beginning of the end. But it is, perhaps, the end of the beginning." Winston Churchill, November 10, 1942
