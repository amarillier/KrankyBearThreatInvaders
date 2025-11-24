# Windows SSH Compilation Setup Guide

## Overview
This guide explains how to set up reliable Windows compilation via SSH from macOS/Linux systems.

## Common Issues and Solutions

### Issue 1: PowerShell Script Parsing Errors
**Symptoms**: Syntax errors when running PowerShell scripts via SSH that work fine in RDP sessions.

**Root Cause**: Line ending differences (LF vs CRLF) and PowerShell execution policy differences between interactive and non-interactive sessions.

**Solution**: Use the `compile-windows.bat` wrapper which:
- Converts LF to CRLF line endings automatically
- Bypasses PowerShell execution policy
- Sets consistent environment

### Issue 2: Execution Policy Restrictions
**Symptoms**: "execution of scripts is disabled" errors.

**Solution**: The batch wrapper uses `-ExecutionPolicy Bypass` to override restrictions.

### Issue 3: Environment Differences
**Symptoms**: Scripts work in RDP but fail via SSH.

**Solution**: The batch wrapper uses `-NoProfile` to ensure consistent environment.

## Recommended Workflow

### Initial Setup (One-time)
1. **First-time setup**: RDP to Windows machine and run:
   ```powershell
   .\compile-windows.ps1 -Windows -Package
   ```
   This ensures all dependencies are set up correctly.

2. **Verify SSH access**: From macOS/Linux:
   ```bash
   ./compile-all.sh
   ```

### Daily Workflow
1. **From macOS/Linux**: Run `./compile-all.sh` which will:
   - Sync files to Windows
   - Execute compilation via SSH
   - Sync results back

2. **If issues occur**: RDP to Windows and run the script directly once, then SSH will work again.

## File Structure

```
compile-windows.bat      # Batch wrapper (fixes line endings, execution policy)
compile-windows.ps1      # Main PowerShell compilation script
prepare-deps.ps1         # Dependency preparation script
check-windows-env.ps1    # Pre-flight environment check script
compile-windows-ssh.sh    # SSH invocation script (macOS/Linux)
sync2windows.sh          # File sync script
```

## Troubleshooting

### Script fails via SSH but works in RDP
1. RDP to Windows
2. Run: `.\compile-windows.ps1 -Windows`
3. This "primes" the environment
4. SSH should now work

### Line ending issues
The batch wrapper automatically fixes line endings. If issues persist:
1. Check file encoding (should be UTF-8)
2. Ensure batch wrapper is being called (not PowerShell directly)
3. Verify `compile-windows.bat` exists and is executable

### PowerShell execution policy
If you see execution policy errors:
1. The batch wrapper should handle this automatically
2. If not, manually set: `Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser`

## Best Practices

1. **Always use the batch wrapper** when invoking via SSH
2. **Run once via RDP** after major changes to "prime" the environment
3. **Keep scripts in sync** between macOS and Windows
4. **Use consistent line endings** - let the batch wrapper handle conversion
5. **Test locally first** - RDP to Windows and test before relying on SSH

## Future Improvements

Consider adding:
- Pre-flight checks script (verify Go, dependencies, etc.) - ✓ Already added as check-windows-env.ps1
- Better error logging to file
- Automatic retry logic
- Environment validation before compilation
- Dependency verification script

