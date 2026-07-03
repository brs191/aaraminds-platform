<#
.SYNOPSIS
  Wire AaraMinds skills into .claude/skills/ so Claude Code can discover them.

.DESCRIPTION
  Claude Code discovers skills by scanning <workspace>/.claude/skills/<name>/SKILL.md.
  The AaraMinds skills live in two canonical homes that are NOT under that path:
    - skills-pack/.claude/skills/   (26 engineering skills)
    - instruction-os/skills/        (3 communication skills)
  This script creates a Windows directory JUNCTION in .claude/skills/ for each one,
  pointing at its canonical home. Junctions need no admin rights / developer mode
  (unlike symbolic links) and are followed transparently by the skill scanner.
  Sources are enumerated from disk, so new skills are picked up automatically on re-run.

.PARAMETER Unwire
  Remove every junction this script created. Real directories and the canonical
  source folders are left untouched.

.EXAMPLE
  pwsh .claude/wire-skills.ps1            # wire (or refresh) all skills
  pwsh .claude/wire-skills.ps1 -Unwire    # remove all junctions

.NOTES
  Re-runnable. Junctions are local wiring and should NOT be committed (see .gitignore).
  After running, restart Claude Code / reopen the workspace and run /skills to confirm.
#>
[CmdletBinding()]
param([switch]$Unwire)

$ErrorActionPreference = 'Stop'
$repo = Split-Path -Parent $PSScriptRoot          # .claude -> workspace root
$dest = Join-Path $PSScriptRoot 'skills'          # .claude/skills
$sources = @(
    (Join-Path $repo 'skills-pack\.claude\skills'),
    (Join-Path $repo 'instruction-os\skills')
)

New-Item -ItemType Directory -Force -Path $dest | Out-Null

function Test-IsReparsePoint([string]$path) {
    (Get-Item -LiteralPath $path -Force).Attributes -band [IO.FileAttributes]::ReparsePoint
}

if ($Unwire) {
    $removed = 0
    Get-ChildItem -LiteralPath $dest -Directory -Force | ForEach-Object {
        if (Test-IsReparsePoint $_.FullName) {
            cmd /c rmdir "$($_.FullName)" | Out-Null
            Write-Host "unlinked  $($_.Name)"
            $removed++
        }
    }
    Write-Host ""
    Write-Host "Unwired. Removed $removed junction(s) from .claude/skills/."
    return
}

$linked = 0; $skipped = 0
foreach ($src in $sources) {
    if (-not (Test-Path -LiteralPath $src)) { Write-Warning "source missing: $src"; continue }
    Get-ChildItem -LiteralPath $src -Directory | ForEach-Object {
        $name   = $_.Name
        $target = $_.FullName
        $link   = Join-Path $dest $name
        if (Test-Path -LiteralPath $link) {
            if (Test-IsReparsePoint $link) {
                cmd /c rmdir "$link" | Out-Null          # refresh existing junction
            } else {
                Write-Warning "skip (real directory, not a junction): $name"; $skipped++; return
            }
        }
        cmd /c mklink /J "$link" "$target" | Out-Null
        Write-Host "linked    $name"
        $linked++
    }
}
Write-Host ""
Write-Host "Done. Linked $linked skill(s) into .claude/skills/ ($skipped skipped)."
Write-Host "Restart Claude Code / reopen the workspace, then run /skills to confirm discovery."
