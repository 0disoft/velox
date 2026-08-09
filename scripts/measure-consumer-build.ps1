[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string] $Cli,

    [Parameter(Mandatory = $true)]
    [string] $WorkRoot,

    [Parameter(Mandatory = $true)]
    [string] $ResultPath,

    [string] $SchemaPath = 'schema/consumer-benchmark-v1.schema.json',

    [ValidateRange(3, 100)]
    [int] $Repetitions = 10,

    [ValidateSet('hello', 'asset-pack')]
    [string] $FixtureKind = 'hello',

    [switch] $Enforce
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

function Invoke-VeloxJson {
    param([string[]] $Arguments)

    $stderrPath = Join-Path $sessionRoot ('stderr-' + [guid]::NewGuid().ToString('N') + '.txt')
    $stdoutLines = & $Cli @Arguments 2> $stderrPath
    $exitCode = $LASTEXITCODE
    $stderrText = if (Test-Path -LiteralPath $stderrPath) {
        [System.IO.File]::ReadAllText($stderrPath)
    } else {
        ''
    }
    Remove-Item -LiteralPath $stderrPath -Force -ErrorAction SilentlyContinue
    if ($exitCode -ne 0) {
        throw "Velox exited with code ${exitCode}: $stderrText"
    }
    $json = ($stdoutLines -join [Environment]::NewLine)
    if ([string]::IsNullOrWhiteSpace($json)) {
        throw 'Velox returned no JSON output.'
    }
    $value = $json | ConvertFrom-Json
    if (-not $value.ok) {
        throw "Velox returned a failure envelope for $($Arguments[0])."
    }
    return $value
}

function Get-FileSnapshot {
    param([string] $Root)

    $snapshot = [ordered]@{}
    if (-not (Test-Path -LiteralPath $Root)) {
        return $snapshot
    }
    foreach ($file in Get-ChildItem -LiteralPath $Root -Recurse -Force -File | Sort-Object FullName) {
        $relative = [System.IO.Path]::GetRelativePath($Root, $file.FullName).Replace('\', '/')
        $snapshot[$relative] = [ordered]@{
            bytes = [int64] $file.Length
            sha256 = (Get-FileHash -LiteralPath $file.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
        }
    }
    return $snapshot
}

function Compare-Snapshot {
    param($Before, $After)

    $changes = [System.Collections.Generic.List[string]]::new()
    $keys = @($Before.Keys) + @($After.Keys) | Sort-Object -Unique
    foreach ($key in $keys) {
        if (-not $Before.Contains($key) -or -not $After.Contains($key)) {
            $changes.Add($key)
            continue
        }
        if ($Before[$key].bytes -ne $After[$key].bytes -or $Before[$key].sha256 -ne $After[$key].sha256) {
            $changes.Add($key)
        }
    }
    return @($changes)
}

function Get-DirectoryBytes {
    param([string] $Path)

    if (-not (Test-Path -LiteralPath $Path)) {
        return [int64] 0
    }
    return [int64] ((Get-ChildItem -LiteralPath $Path -Recurse -Force -File | Measure-Object -Property Length -Sum).Sum ?? 0)
}

function Get-Percentile {
    param([double[]] $Values, [double] $Percentile)

    $sorted = @($Values | Sort-Object)
    $index = [Math]::Max(0, [Math]::Ceiling($Percentile * $sorted.Count) - 1)
    return [Math]::Round([double] $sorted[$index], 3)
}

function New-AssetPackFixture {
    param([string] $Root)

    $seed = [uint32] 1447383631
    $fileCount = 1000
    $totalBytes = 10485760
    $baseSize = [Math]::Floor($totalBytes / $fileCount)
    $remainder = $totalBytes % $fileCount
    $expectedTreeSha256 = '8b7fb697154d4fb91ae7a4f9c797109ff2b83f25cad8d0d947b9f523ef83005f'
    $treeHash = [System.Security.Cryptography.IncrementalHash]::CreateHash([System.Security.Cryptography.HashAlgorithmName]::SHA256)
    $utf8 = [System.Text.UTF8Encoding]::new($false)

    try {
        for ($index = 0; $index -lt $fileCount; $index++) {
            $directory = ([int] [Math]::Floor($index / 100)).ToString('D3')
            $relative = 'assets/{0}/{1}.bin' -f $directory, $index.ToString('D6')
            $destination = Join-Path $Root ($relative.Replace('/', [System.IO.Path]::DirectorySeparatorChar))
            [System.IO.Directory]::CreateDirectory([System.IO.Path]::GetDirectoryName($destination)) | Out-Null

            $size = [int] ($baseSize + $(if ($index -lt $remainder) { 1 } else { 0 }))
            $bytes = [byte[]]::new($size)
            $mixed = [uint32] (([uint64] ($index + 1) * [uint64] 2654435769) -band [uint64] 4294967295)
            $state = [uint32] ($seed -bxor $mixed)
            $state = [uint32] (([uint64] ($state -bxor ($state -shr 16))) -band [uint64] 4294967295)
            $state = [uint32] (([uint64] $state * [uint64] 2246822507) -band [uint64] 4294967295)
            $state = [uint32] (([uint64] ($state -bxor ($state -shr 13))) -band [uint64] 4294967295)
            $state = [uint32] (([uint64] $state * [uint64] 3266489909) -band [uint64] 4294967295)
            $state = [uint32] (([uint64] ($state -bxor ($state -shr 16))) -band [uint64] 4294967295)
            if ($state -eq 0) {
                $state = [uint32] 1831565813
            }
            for ($offset = 0; $offset -lt $bytes.Length; $offset++) {
                $state = [uint32] (([uint64] ($state -bxor ($state -shl 13))) -band [uint64] 4294967295)
                $state = [uint32] (([uint64] ($state -bxor ($state -shr 17))) -band [uint64] 4294967295)
                $state = [uint32] (([uint64] ($state -bxor ($state -shl 5))) -band [uint64] 4294967295)
                $bytes[$offset] = [byte] ($state -band 0xff)
            }

            [System.IO.File]::WriteAllBytes($destination, $bytes)
            $treeHash.AppendData($utf8.GetBytes($relative))
            $treeHash.AppendData([byte[]] @(0))
            $treeHash.AppendData([System.BitConverter]::GetBytes([uint64] $bytes.Length))
            $treeHash.AppendData($bytes)
        }
        $treeSha256 = [System.Convert]::ToHexString($treeHash.GetHashAndReset()).ToLowerInvariant()
    } finally {
        $treeHash.Dispose()
    }
    if ($treeSha256 -ne $expectedTreeSha256) {
        throw "Asset-pack generator digest mismatch: $treeSha256"
    }
    return [pscustomobject]@{
        files = $fileCount
        bytes = [int64] $totalBytes
        treeSha256 = $treeSha256
    }
}

function Get-ProcessTrace {
    param($Events, [int] $ParentPid, [string] $CliName, [bool] $Available)

    if (-not $Available) {
        return [pscustomobject]@{
            complete = $false
            rootProcessCount = 0
            descendants = @()
            forbiddenDescendants = @()
        }
    }

    $records = foreach ($eventItem in $Events) {
        $native = $eventItem.SourceEventArgs.NewEvent
        [pscustomobject]@{
            pid = [int] $native.ProcessID
            parentPid = [int] $native.ParentProcessID
            name = [string] $native.ProcessName
        }
    }
    $rootProcesses = @($records | Where-Object {
        $_.parentPid -eq $ParentPid -and $_.name -ieq $CliName
    })
    $known = [System.Collections.Generic.HashSet[int]]::new()
    foreach ($record in $rootProcesses) {
        [void] $known.Add($record.pid)
    }
    $descendants = [System.Collections.Generic.List[object]]::new()
    $changed = $true
    while ($changed) {
        $changed = $false
        foreach ($record in $records) {
            if ($known.Contains($record.parentPid) -and -not $known.Contains($record.pid)) {
                [void] $known.Add($record.pid)
                $descendants.Add($record)
                $changed = $true
            }
        }
    }
    $forbiddenPattern = '^(go|node|npm|npx|pnpm|yarn|bun|cargo|rustc|zig|cl|link|cmake|ninja|msbuild)(\.exe|\.cmd|\.bat)?$'
    return [pscustomobject]@{
        complete = $rootProcesses.Count -eq 1
        rootProcessCount = $rootProcesses.Count
        descendants = @($descendants | Sort-Object pid)
        forbiddenDescendants = @($descendants | Where-Object { $_.name -match $forbiddenPattern } | Sort-Object pid)
    }
}

$Cli = [System.IO.Path]::GetFullPath($Cli)
$WorkRoot = [System.IO.Path]::GetFullPath($WorkRoot)
$ResultPath = [System.IO.Path]::GetFullPath($ResultPath)
$SchemaPath = [System.IO.Path]::GetFullPath($SchemaPath)
if (-not (Test-Path -LiteralPath $Cli -PathType Leaf)) {
    throw "Velox CLI is missing: $Cli"
}
if (-not (Test-Path -LiteralPath $SchemaPath -PathType Leaf)) {
    throw "Benchmark schema is missing: $SchemaPath"
}

$sessionRoot = Join-Path $WorkRoot ('run-' + [DateTime]::UtcNow.ToString('yyyyMMddTHHmmssfffZ') + '-' + [guid]::NewGuid().ToString('N'))
$projectRoot = Join-Path $sessionRoot 'project'
$outputsRoot = Join-Path $sessionRoot 'outputs'
New-Item -ItemType Directory -Path $sessionRoot, $outputsRoot -Force | Out-Null

$initialized = Invoke-VeloxJson -Arguments @('init', $projectRoot, '--json')
$configPath = Join-Path $projectRoot 'velox.json'
$appId = [string] $initialized.result.appId
$appKey = $appId
$generatedFixture = if ($FixtureKind -eq 'asset-pack') {
    New-AssetPackFixture -Root (Join-Path $projectRoot 'web')
} else {
    [pscustomobject]@{ files = 0; bytes = [int64] 0; treeSha256 = $null }
}
$validation = Invoke-VeloxJson -Arguments @('validate', '--config', $configPath, '--json')
$projectBefore = Get-FileSnapshot -Root $projectRoot

$cachePaths = @(
    (Join-Path $projectRoot '.cache'),
    (Join-Path $projectRoot '.velox'),
    (Join-Path $projectRoot 'node_modules'),
    (Join-Path $projectRoot 'target')
)
$cacheBefore = [int64] 0
foreach ($cachePath in $cachePaths) {
    $cacheBefore += Get-DirectoryBytes -Path $cachePath
}

$samples = [System.Collections.Generic.List[object]]::new()
for ($index = 0; $index -lt $Repetitions; $index++) {
    $outputRoot = Join-Path $outputsRoot $index.ToString('D2')
    $stderrPath = Join-Path $sessionRoot ('build-' + $index.ToString('D2') + '.stderr.txt')
    $sourceIdentifier = 'velox-consumer-bench-' + [guid]::NewGuid().ToString('N')
    $traceAvailable = $true
    try {
        Register-WmiEvent -Class Win32_ProcessStartTrace -SourceIdentifier $sourceIdentifier -ErrorAction Stop | Out-Null
    } catch {
        try {
            Register-CimIndicationEvent -ClassName Win32_ProcessStartTrace -SourceIdentifier $sourceIdentifier -ErrorAction Stop | Out-Null
        } catch {
            $traceAvailable = $false
        }
    }
    try {
        $previousBenchMode = $env:VELOX_BENCH_MODE
        $previousBenchPhases = $env:VELOX_BENCH_PHASES
        $env:VELOX_BENCH_MODE = '1'
        $env:VELOX_BENCH_PHASES = '1'
        $timer = [System.Diagnostics.Stopwatch]::StartNew()
        $stdoutLines = & $Cli build --config $configPath --out $outputRoot --json 2> $stderrPath
        $exitCode = $LASTEXITCODE
        $timer.Stop()
        if ($traceAvailable) {
            [void] (Wait-Event -SourceIdentifier $sourceIdentifier -Timeout 1)
            Start-Sleep -Milliseconds 250
            $events = @(Get-Event -SourceIdentifier $sourceIdentifier -ErrorAction SilentlyContinue)
        } else {
            $events = @()
        }
    } finally {
        if ($null -eq $previousBenchMode) { Remove-Item Env:VELOX_BENCH_MODE -ErrorAction SilentlyContinue } else { $env:VELOX_BENCH_MODE = $previousBenchMode }
        if ($null -eq $previousBenchPhases) { Remove-Item Env:VELOX_BENCH_PHASES -ErrorAction SilentlyContinue } else { $env:VELOX_BENCH_PHASES = $previousBenchPhases }
        if ($traceAvailable) {
            Unregister-Event -SourceIdentifier $sourceIdentifier -ErrorAction SilentlyContinue
            Remove-Event -SourceIdentifier $sourceIdentifier -ErrorAction SilentlyContinue
        }
    }
    $stderrText = if (Test-Path -LiteralPath $stderrPath) { [System.IO.File]::ReadAllText($stderrPath) } else { '' }
    if ($exitCode -ne 0) {
        throw "Measured build $index exited with code ${exitCode}: $stderrText"
    }
    $phases = @(
        foreach ($line in ($stderrText -split '\r?\n')) {
            if ($line -match '^VELOX_PHASE ([a-z.]+) ([0-9]+)$') {
                [pscustomobject]@{ name = $Matches[1]; durationUs = [int64] $Matches[2] }
            }
        }
    )
    if ($phases.Count -eq 0) {
        throw "Measured build $index returned no phase evidence."
    }
    $envelope = (($stdoutLines -join [Environment]::NewLine) | ConvertFrom-Json)
    if (-not $envelope.ok) {
        throw "Measured build $index returned a failure envelope."
    }
    $archivePath = Join-Path $outputRoot ($appKey + '.zip')
    $directoryPath = Join-Path $outputRoot $appKey
    $inspection = Invoke-VeloxJson -Arguments @('inspect', $archivePath, '--json')
    $topLevel = @(Get-ChildItem -LiteralPath $outputRoot -Force)
    $unexpected = @($topLevel | Where-Object {
        $_.FullName -ne $directoryPath -and $_.FullName -ne $archivePath
    })
    $trace = Get-ProcessTrace -Events $events -ParentPid $PID -CliName ([System.IO.Path]::GetFileName($Cli)) -Available $traceAvailable
    $samples.Add([pscustomobject]@{
        index = $index
        outcome = 'success'
        durationMs = [Math]::Round($timer.Elapsed.TotalMilliseconds, 3)
        archiveBytes = [int64] $envelope.result.archiveBytes
        archiveSha256 = [string] $envelope.result.archiveSha256
        portableFiles = [int] $inspection.result.portableFiles
        portableBytes = [int64] $inspection.result.portableBytes
        survivingIntermediateFiles = $unexpected.Count
        survivingIntermediatePaths = @($unexpected | ForEach-Object { $_.Name } | Sort-Object)
        phases = $phases
        processTrace = $trace
    })
}

$cacheAfter = [int64] 0
foreach ($cachePath in $cachePaths) {
    $cacheAfter += Get-DirectoryBytes -Path $cachePath
}
$projectAfter = Get-FileSnapshot -Root $projectRoot
$projectChanges = @(Compare-Snapshot -Before $projectBefore -After $projectAfter)
$durations = [double[]] @($samples | ForEach-Object { $_.durationMs })
$allTraceComplete = @($samples | Where-Object { -not $_.processTrace.complete }).Count -eq 0
$forbiddenProcesses = @($samples | ForEach-Object { $_.processTrace.forbiddenDescendants })
$intermediateCount = [int] (($samples | Measure-Object -Property survivingIntermediateFiles -Sum).Sum ?? 0)
$cacheDelta = [Math]::Max([int64] 0, $cacheAfter - $cacheBefore)

$processStatus = if (-not $allTraceComplete) { 'unverified' } elseif ($forbiddenProcesses.Count -gt 0) { 'fail' } else { 'pass' }
$p95 = Get-Percentile -Values $durations -Percentile 0.95
$result = [ordered]@{
    schemaVersion = 'velox.consumer-benchmark/v1'
    scope = 'local-clean-output-build-command'
    evidenceLevel = 'controlled-local-observation'
    releaseVersion = [string] (Invoke-VeloxJson -Arguments @('version', '--json')).result.version
    repetitions = $Repetitions
    measurement = [ordered]@{
        tool = 'scripts/measure-consumer-build.ps1'
        toolVersion = 3
        metric = 'build-command-duration'
        unit = 'milliseconds'
        clock = 'System.Diagnostics.Stopwatch'
        startsAt = 'immediately-before-velox-process-launch'
        endsAt = 'velox-process-exit'
        warmupSamples = 0
        concurrency = 1
        projectState = 'same-initialized-project'
        outputState = 'new-empty-output-root-per-sample'
        osFileCacheState = 'uncontrolled-and-potentially-warm-after-first-sample'
        excludedWork = @('project-initialization', 'project-validation', 'artifact-inspection', 'schema-validation', 'process-trace-drain')
    }
    fixture = [ordered]@{
        kind = if ($FixtureKind -eq 'asset-pack') { 'deterministic-asset-pack' } else { 'dependency-free-init-template' }
        appId = $appId
        assetFiles = [int] $validation.result.assetFiles
        assetBytes = [int64] $validation.result.assetBytes
        assetSha256 = [string] $validation.result.assetSha256
        generatedFiles = [int] $generatedFixture.files
        generatedBytes = [int64] $generatedFixture.bytes
        generatedTreeSha256 = $generatedFixture.treeSha256
    }
    environment = [ordered]@{
        os = [Environment]::OSVersion.VersionString
        architecture = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
        runnerImage = $env:ImageOS
        runnerImageVersion = $env:ImageVersion
    }
    summary = [ordered]@{
        minMs = [Math]::Round(($durations | Measure-Object -Minimum).Minimum, 3)
        p50Ms = Get-Percentile -Values $durations -Percentile 0.50
        p95Ms = $p95
        maxMs = [Math]::Round(($durations | Measure-Object -Maximum).Maximum, 3)
    }
    cache = [ordered]@{
        workflowDeclaredActionsCacheUploadBytes = 0
        veloxOwnedBytesBefore = $cacheBefore
        veloxOwnedBytesAfter = $cacheAfter
        veloxOwnedDeltaBytes = $cacheDelta
        paths = @($cachePaths | ForEach-Object { [System.IO.Path]::GetRelativePath($projectRoot, $_).Replace('\', '/') })
    }
    workspace = [ordered]@{
        sourceChangesOutsideOutput = @($projectChanges)
        survivingIntermediateFiles = $intermediateCount
    }
    gates = [ordered]@{
        p95AtOrBelowTwoSeconds = if ($FixtureKind -eq 'asset-pack') { 'not-applicable' } elseif ($p95 -le 2000) { 'pass' } else { 'fail' }
        zeroVeloxCacheGrowth = if ($cacheDelta -eq 0) { 'pass' } else { 'fail' }
        zeroSurvivingIntermediates = if ($intermediateCount -eq 0 -and $projectChanges.Count -eq 0) { 'pass' } else { 'fail' }
        noCompilerOrPackageManagerChildren = $processStatus
    }
    samples = @($samples)
}

$resultDirectory = [System.IO.Path]::GetDirectoryName($ResultPath)
New-Item -ItemType Directory -Path $resultDirectory -Force | Out-Null
$json = $result | ConvertTo-Json -Depth 12
if (-not ($json | Test-Json -SchemaFile $SchemaPath -ErrorAction Stop)) {
    throw 'Consumer benchmark result does not satisfy its JSON schema.'
}
[System.IO.File]::WriteAllText($ResultPath, $json + [Environment]::NewLine, [System.Text.UTF8Encoding]::new($false))
$json

if ($Enforce) {
    $failed = @($result.gates.GetEnumerator() | Where-Object { $_.Value -ne 'pass' })
    if ($failed.Count -gt 0) {
        exit 1
    }
}
