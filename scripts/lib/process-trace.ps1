Set-StrictMode -Version Latest

function Start-VeloxProcessTrace {
    $sourceIdentifier = 'velox-process-trace-' + [guid]::NewGuid().ToString('N')
    try {
        Register-WmiEvent -Class Win32_ProcessStartTrace -SourceIdentifier $sourceIdentifier -ErrorAction Stop | Out-Null
        return [pscustomobject]@{ SourceIdentifier = $sourceIdentifier; Available = $true }
    } catch {
        try {
            Register-CimIndicationEvent -ClassName Win32_ProcessStartTrace -SourceIdentifier $sourceIdentifier -ErrorAction Stop | Out-Null
            return [pscustomobject]@{ SourceIdentifier = $sourceIdentifier; Available = $true }
        } catch {
            return [pscustomobject]@{ SourceIdentifier = $sourceIdentifier; Available = $false }
        }
    }
}

function Complete-VeloxProcessTrace {
    param(
        [Parameter(Mandatory = $true)]
        $Trace,

        [Parameter(Mandatory = $true)]
        [int] $ParentPid,

        [Parameter(Mandatory = $true)]
        [string] $RootProcessName
    )

    if (-not $Trace.Available) {
        return [pscustomobject]@{
            status = 'unverified'
            rootProcessCount = 0
            descendants = @()
            forbiddenDescendants = @()
        }
    }

    try {
        [void] (Wait-Event -SourceIdentifier $Trace.SourceIdentifier -Timeout 1)
        Start-Sleep -Milliseconds 250
        $events = @(Get-Event -SourceIdentifier $Trace.SourceIdentifier -ErrorAction SilentlyContinue)
    } finally {
        Unregister-Event -SourceIdentifier $Trace.SourceIdentifier -ErrorAction SilentlyContinue
        Remove-Event -SourceIdentifier $Trace.SourceIdentifier -ErrorAction SilentlyContinue
    }

    $records = foreach ($eventItem in $events) {
        $native = $eventItem.SourceEventArgs.NewEvent
        [pscustomobject]@{
            pid = [int] $native.ProcessID
            parentPid = [int] $native.ParentProcessID
            name = [string] $native.ProcessName
        }
    }
    $roots = @($records | Where-Object {
        $_.parentPid -eq $ParentPid -and $_.name -ieq $RootProcessName
    })
    $known = [System.Collections.Generic.HashSet[int]]::new()
    foreach ($root in $roots) {
        [void] $known.Add($root.pid)
    }
    $descendantRecords = [System.Collections.Generic.List[object]]::new()
    $changed = $true
    while ($changed) {
        $changed = $false
        foreach ($record in $records) {
            if ($known.Contains($record.parentPid) -and -not $known.Contains($record.pid)) {
                [void] $known.Add($record.pid)
                $descendantRecords.Add($record)
                $changed = $true
            }
        }
    }

    $forbiddenPattern = '^(go|node|npm|npx|pnpm|yarn|bun|cargo|rustc|zig|cl|link|cmake|ninja|msbuild)(\.exe|\.cmd|\.bat)?$'
    $descendants = @($descendantRecords | ForEach-Object name | Sort-Object -Unique)
    $forbidden = @($descendants | Where-Object { $_ -match $forbiddenPattern })
    $status = if ($roots.Count -ne 1) {
        'unverified'
    } elseif ($forbidden.Count -gt 0) {
        'fail'
    } else {
        'pass'
    }
    return [pscustomobject]@{
        status = $status
        rootProcessCount = $roots.Count
        descendants = $descendants
        forbiddenDescendants = $forbidden
    }
}
