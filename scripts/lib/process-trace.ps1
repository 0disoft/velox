Set-StrictMode -Version Latest

function Initialize-VeloxSnapshotPoller {
    if ('Velox.ProcessTrace.SnapshotPoller' -as [type]) {
        return
    }
    Add-Type -TypeDefinition @'
using System;
using System.Collections.Generic;
using System.Runtime.InteropServices;
using System.Threading;

namespace Velox.ProcessTrace {
    public sealed class ProcessRecord {
        public int pid;
        public int parentPid;
        public string name = "";
    }

    public sealed class SnapshotPoller : IDisposable {
        const uint TH32CS_SNAPPROCESS = 0x00000002;
        static readonly IntPtr InvalidHandle = new IntPtr(-1);
        readonly object gate = new object();
        readonly HashSet<int> known = new HashSet<int>();
        readonly List<ProcessRecord> records = new List<ProcessRecord>();
        Thread thread;
        volatile bool stopping;

        [StructLayout(LayoutKind.Sequential, CharSet = CharSet.Unicode)]
        struct PROCESSENTRY32 {
            public uint dwSize;
            public uint cntUsage;
            public uint th32ProcessID;
            public IntPtr th32DefaultHeapID;
            public uint th32ModuleID;
            public uint cntThreads;
            public uint th32ParentProcessID;
            public int pcPriClassBase;
            public uint dwFlags;
            [MarshalAs(UnmanagedType.ByValTStr, SizeConst = 260)]
            public string szExeFile;
        }

        [DllImport("kernel32.dll", SetLastError = true)]
        static extern IntPtr CreateToolhelp32Snapshot(uint flags, uint processId);
        [DllImport("kernel32.dll", CharSet = CharSet.Unicode, SetLastError = true)]
        static extern bool Process32FirstW(IntPtr snapshot, ref PROCESSENTRY32 entry);
        [DllImport("kernel32.dll", CharSet = CharSet.Unicode, SetLastError = true)]
        static extern bool Process32NextW(IntPtr snapshot, ref PROCESSENTRY32 entry);
        [DllImport("kernel32.dll")]
        static extern bool CloseHandle(IntPtr handle);

        public void Start() {
            foreach (var item in Snapshot()) known.Add(item.pid);
            thread = new Thread(Poll) { IsBackground = true, Name = "velox-process-trace" };
            thread.Start();
        }

        void Poll() {
            while (!stopping) {
                foreach (var item in Snapshot()) {
                    lock (gate) {
                        if (known.Add(item.pid)) records.Add(item);
                    }
                }
                Thread.Sleep(2);
            }
        }

        public ProcessRecord[] Stop() {
            stopping = true;
            if (thread != null) thread.Join();
            lock (gate) return records.ToArray();
        }

        static List<ProcessRecord> Snapshot() {
            var result = new List<ProcessRecord>();
            IntPtr snapshot = CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0);
            if (snapshot == InvalidHandle) return result;
            try {
                var entry = new PROCESSENTRY32 { dwSize = (uint)Marshal.SizeOf(typeof(PROCESSENTRY32)) };
                if (!Process32FirstW(snapshot, ref entry)) return result;
                do {
                    result.Add(new ProcessRecord {
                        pid = unchecked((int)entry.th32ProcessID),
                        parentPid = unchecked((int)entry.th32ParentProcessID),
                        name = entry.szExeFile ?? ""
                    });
                } while (Process32NextW(snapshot, ref entry));
                return result;
            } finally {
                CloseHandle(snapshot);
            }
        }

        public void Dispose() { Stop(); }
    }
}
'@
}

function Start-VeloxProcessTrace {
    $sourceIdentifier = 'velox-process-trace-' + [guid]::NewGuid().ToString('N')
    if ($env:VELOX_PROCESS_TRACE_BACKEND -ne 'snapshot-poller') {
        try {
            Register-WmiEvent -Class Win32_ProcessStartTrace -SourceIdentifier $sourceIdentifier -ErrorAction Stop | Out-Null
            return [pscustomobject]@{ SourceIdentifier = $sourceIdentifier; Available = $true; Watcher = $null; Poller = $null }
        } catch {
            try {
                Register-CimIndicationEvent -ClassName Win32_ProcessStartTrace -SourceIdentifier $sourceIdentifier -ErrorAction Stop | Out-Null
                return [pscustomobject]@{ SourceIdentifier = $sourceIdentifier; Available = $true; Watcher = $null; Poller = $null }
            } catch {
                # PowerShell 7 runner images may not expose either registration cmdlet.
            }
        }
    }

    $watcher = $null
    try {
        $query = [System.Management.WqlEventQuery]::new('SELECT * FROM Win32_ProcessStartTrace')
        $watcher = [System.Management.ManagementEventWatcher]::new($query)
        Register-ObjectEvent -InputObject $watcher -EventName EventArrived -SourceIdentifier $sourceIdentifier -ErrorAction Stop | Out-Null
        $watcher.Start()
        return [pscustomobject]@{ SourceIdentifier = $sourceIdentifier; Available = $true; Watcher = $watcher; Poller = $null }
    } catch {
        if ($null -ne $watcher) {
            $watcher.Dispose()
        }
        Unregister-Event -SourceIdentifier $sourceIdentifier -ErrorAction SilentlyContinue
        Remove-Event -SourceIdentifier $sourceIdentifier -ErrorAction SilentlyContinue
        try {
            Initialize-VeloxSnapshotPoller
            $poller = [Velox.ProcessTrace.SnapshotPoller]::new()
            $poller.Start()
            return [pscustomobject]@{ SourceIdentifier = $sourceIdentifier; Available = $true; Watcher = $null; Poller = $poller }
        } catch {
            return [pscustomobject]@{ SourceIdentifier = $sourceIdentifier; Available = $false; Watcher = $null; Poller = $null }
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

    if ($null -ne $Trace.Poller) {
        $records = @($Trace.Poller.Stop())
        $Trace.Poller.Dispose()
    } else {
        try {
            [void] (Wait-Event -SourceIdentifier $Trace.SourceIdentifier -Timeout 1)
            Start-Sleep -Milliseconds 250
            $events = @(Get-Event -SourceIdentifier $Trace.SourceIdentifier -ErrorAction SilentlyContinue)
        } finally {
            if ($null -ne $Trace.Watcher) {
                try { $Trace.Watcher.Stop() } catch { }
                $Trace.Watcher.Dispose()
            }
            Unregister-Event -SourceIdentifier $Trace.SourceIdentifier -ErrorAction SilentlyContinue
            Remove-Event -SourceIdentifier $Trace.SourceIdentifier -ErrorAction SilentlyContinue
        }

        $records = @($events | ForEach-Object {
            $native = $_.SourceEventArgs.NewEvent
            [pscustomobject]@{
                pid = [int] $native.ProcessID
                parentPid = [int] $native.ParentProcessID
                name = [string] $native.ProcessName
            }
        })
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
