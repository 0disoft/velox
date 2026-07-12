[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string] $ReleaseArchive,

    [Parameter(Mandatory = $true)]
    [string] $WorkRoot,

    [Parameter(Mandatory = $true)]
    [string] $ResultPath,

    [Parameter(Mandatory = $true)]
    [long] $StartedAtUtcTicks,

    [ValidateSet('local-file', 'github-actions-artifact')]
    [string] $AcquisitionMode = 'local-file',

    [string] $ExpectedReleaseSha256 = '',

    [string] $SampleId = 'local',

    [string] $SchemaPath = 'schema/consumer-e2e-v1.schema.json'
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
. (Join-Path $PSScriptRoot 'lib/process-trace.ps1')

function Write-Result {
    param($Value)

    $directory = [System.IO.Path]::GetDirectoryName($ResultPath)
    New-Item -ItemType Directory -Path $directory -Force | Out-Null
    $json = $Value | ConvertTo-Json -Depth 12
    if (-not ($json | Test-Json -SchemaFile $SchemaPath -ErrorAction Stop)) {
        throw 'Consumer end-to-end result does not satisfy its JSON schema.'
    }
    [System.IO.File]::WriteAllText($ResultPath, $json + [Environment]::NewLine, [System.Text.UTF8Encoding]::new($false))
    $json
}

function Invoke-VeloxJson {
    param(
        [string] $Executable,
        [string[]] $Arguments,
        [string] $StderrPath
    )

    $stdout = & $Executable @Arguments 2> $StderrPath
    $exitCode = $LASTEXITCODE
    if ($exitCode -ne 0) {
        throw "Velox exited with code ${exitCode}."
    }
    $json = $stdout -join [Environment]::NewLine
    if ([string]::IsNullOrWhiteSpace($json)) {
        throw 'Velox returned no JSON output.'
    }
    $value = $json | ConvertFrom-Json
    if (-not $value.ok) {
        throw "Velox returned a failure envelope for $($Arguments[0])."
    }
    return $value
}

function Invoke-VeloxTracedJson {
    param(
        [string] $Executable,
        [string[]] $Arguments,
        [string] $StderrPath,
        $Trace
    )

    $startInfo = [System.Diagnostics.ProcessStartInfo]::new()
    $startInfo.FileName = $Executable
    $startInfo.UseShellExecute = $false
    $startInfo.CreateNoWindow = $true
    $startInfo.RedirectStandardOutput = $true
    $startInfo.RedirectStandardError = $true
    foreach ($argument in $Arguments) {
        [void] $startInfo.ArgumentList.Add($argument)
    }
    $process = [System.Diagnostics.Process]::new()
    $process.StartInfo = $startInfo
    try {
        if (-not $process.Start()) {
            throw 'Velox process did not start.'
        }
        $Trace.ManualRecords.Add([pscustomobject]@{
            pid = [int] $process.Id
            parentPid = [int] $PID
            name = [System.IO.Path]::GetFileName($Executable)
        })
        $stdoutTask = $process.StandardOutput.ReadToEndAsync()
        $stderrTask = $process.StandardError.ReadToEndAsync()
        $process.WaitForExit()
        $stdout = $stdoutTask.GetAwaiter().GetResult()
        $stderr = $stderrTask.GetAwaiter().GetResult()
        [System.IO.File]::WriteAllText($StderrPath, $stderr, [System.Text.UTF8Encoding]::new($false))
        if ($process.ExitCode -ne 0) {
            throw "Velox exited with code $($process.ExitCode)."
        }
        if ([string]::IsNullOrWhiteSpace($stdout)) {
            throw 'Velox returned no JSON output.'
        }
        $value = $stdout | ConvertFrom-Json
        if (-not $value.ok) {
            throw "Velox returned a failure envelope for $($Arguments[0])."
        }
        return $value
    } finally {
        $process.Dispose()
    }
}

function Expand-ReleaseArchive {
    param([string] $ArchivePath, [string] $Destination)

    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $destinationRoot = [System.IO.Path]::GetFullPath($Destination) + [System.IO.Path]::DirectorySeparatorChar
    $zip = [System.IO.Compression.ZipFile]::OpenRead($ArchivePath)
    try {
        foreach ($entry in $zip.Entries) {
            $entryPath = $entry.FullName.Replace('/', [System.IO.Path]::DirectorySeparatorChar)
            if ([System.IO.Path]::IsPathRooted($entryPath)) {
                throw 'Release archive contains an absolute path.'
            }
            $target = [System.IO.Path]::GetFullPath((Join-Path $Destination $entryPath))
            if (-not $target.StartsWith($destinationRoot, [System.StringComparison]::OrdinalIgnoreCase)) {
                throw 'Release archive contains a path outside its extraction root.'
            }
        }
    } finally {
        $zip.Dispose()
    }
    [System.IO.Compression.ZipFile]::ExtractToDirectory($ArchivePath, $Destination)
}

$ReleaseArchive = [System.IO.Path]::GetFullPath($ReleaseArchive)
$WorkRoot = [System.IO.Path]::GetFullPath($WorkRoot)
$ResultPath = [System.IO.Path]::GetFullPath($ResultPath)
$SchemaPath = [System.IO.Path]::GetFullPath($SchemaPath)
$startedAt = [DateTime]::new($StartedAtUtcTicks, [DateTimeKind]::Utc)
$osFileCacheState = if ($AcquisitionMode -eq 'github-actions-artifact') {
    'fresh-hosted-runner-when-executed-in-an-isolated-job'
} else {
    'uncontrolled-local-state'
}
$phase = 'input-validation'
$hostedGateFailed = $false
$sessionRoot = Join-Path $WorkRoot ('run-' + [DateTime]::UtcNow.ToString('yyyyMMddTHHmmssfffZ') + '-' + [guid]::NewGuid().ToString('N'))
$stderrPath = Join-Path $sessionRoot 'velox.stderr.txt'

try {
    if ($startedAt -gt [DateTime]::UtcNow) {
        throw 'Measurement start is in the future.'
    }
    if ($AcquisitionMode -eq 'github-actions-artifact' -and $env:GITHUB_ACTIONS -ne 'true') {
        throw 'GitHub Actions acquisition mode requires a GitHub Actions runner.'
    }
    if (-not (Test-Path -LiteralPath $ReleaseArchive -PathType Leaf)) {
        throw 'Release archive is missing.'
    }
    if (-not (Test-Path -LiteralPath $SchemaPath -PathType Leaf)) {
        throw 'Consumer end-to-end schema is missing.'
    }
    New-Item -ItemType Directory -Path $sessionRoot -Force | Out-Null

    $phase = 'release-verification'
    $releaseInfo = Get-Item -LiteralPath $ReleaseArchive
    $releaseSha256 = (Get-FileHash -LiteralPath $ReleaseArchive -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($ExpectedReleaseSha256 -and $releaseSha256 -ne $ExpectedReleaseSha256.ToLowerInvariant()) {
        throw 'Release archive checksum does not match the expected digest.'
    }

    $phase = 'release-extraction'
    $releaseRoot = Join-Path $sessionRoot 'release'
    New-Item -ItemType Directory -Path $releaseRoot | Out-Null
    Expand-ReleaseArchive -ArchivePath $ReleaseArchive -Destination $releaseRoot
    $cliFiles = @(Get-ChildItem -LiteralPath $releaseRoot -Recurse -File -Filter 'velox.exe')
    if ($cliFiles.Count -ne 1) {
        throw 'Release archive must contain exactly one velox.exe.'
    }
    $cli = $cliFiles[0].FullName
    $releaseDirectory = [System.IO.Path]::GetDirectoryName($cli)
    foreach ($requiredReleaseFile in @('release-manifest.json', 'schema/velox-v1.schema.json')) {
        if (-not (Test-Path -LiteralPath (Join-Path $releaseDirectory $requiredReleaseFile) -PathType Leaf)) {
            throw "Release archive is missing $requiredReleaseFile."
        }
    }

    $phase = 'project-initialization'
    $projectRoot = Join-Path $sessionRoot 'project'
    $initialized = Invoke-VeloxJson -Executable $cli -Arguments @('init', $projectRoot, '--json') -StderrPath $stderrPath
    $configPath = Join-Path $projectRoot 'velox.json'
    $validated = Invoke-VeloxJson -Executable $cli -Arguments @('validate', '--config', $configPath, '--json') -StderrPath $stderrPath

    $phase = 'consumer-build'
    $outputRoot = Join-Path $sessionRoot 'output'
    $processTraceHandle = Start-VeloxProcessTrace
    $buildTimer = [System.Diagnostics.Stopwatch]::StartNew()
    try {
        $built = Invoke-VeloxTracedJson -Executable $cli -Arguments @('build', '--config', $configPath, '--out', $outputRoot, '--json') -StderrPath $stderrPath -Trace $processTraceHandle
    } catch {
        [void] (Complete-VeloxProcessTrace -Trace $processTraceHandle -ParentPid $PID -RootProcessName ([System.IO.Path]::GetFileName($cli)))
        throw
    } finally {
        $buildTimer.Stop()
    }
    $processTrace = Complete-VeloxProcessTrace -Trace $processTraceHandle -ParentPid $PID -RootProcessName ([System.IO.Path]::GetFileName($cli))

    $phase = 'output-verification'
    $appId = [string] $initialized.result.appId
    $appKey = $appId
    $archivePath = Join-Path $outputRoot ($appKey + '.zip')
    $inspected = Invoke-VeloxJson -Executable $cli -Arguments @('inspect', $archivePath, '--json') -StderrPath $stderrPath
    $unexpected = @(Get-ChildItem -LiteralPath $outputRoot -Force | Where-Object {
        $_.Name -ne $appKey -and $_.Name -ne ($appKey + '.zip')
    })
    $finishedAt = [DateTime]::UtcNow
    $durationMs = [Math]::Round(($finishedAt - $startedAt).TotalMilliseconds, 3)
    $releaseVersion = [string] (Invoke-VeloxJson -Executable $cli -Arguments @('version', '--json') -StderrPath $stderrPath).result.version

    $phase = 'result-validation'
    $result = [ordered]@{
        schemaVersion = 'velox.consumer-e2e/v1'
        scope = 'checkout-complete-to-portable-zip'
        evidenceLevel = if ($AcquisitionMode -eq 'github-actions-artifact') { 'hosted-runner-evidence' } else { 'local-contract-smoke' }
        sampleId = $SampleId
        outcome = 'success'
        startedAtUtc = $startedAt.ToString('o')
        finishedAtUtc = $finishedAt.ToString('o')
        durationMs = $durationMs
        acquisition = [ordered]@{
            mode = $AcquisitionMode
            archiveBytes = [int64] $releaseInfo.Length
            archiveSha256 = $releaseSha256
            expectedSha256Verified = [bool] $ExpectedReleaseSha256
        }
        release = [ordered]@{
            version = $releaseVersion
            target = 'windows-x64'
        }
        fixture = [ordered]@{
            kind = 'dependency-free-init-template'
            appId = $appId
            assetFiles = [int] $validated.result.assetFiles
            assetBytes = [int64] $validated.result.assetBytes
            assetSha256 = [string] $validated.result.assetSha256
        }
        build = [ordered]@{
            durationMs = [Math]::Round($buildTimer.Elapsed.TotalMilliseconds, 3)
            archiveBytes = [int64] $built.result.archiveBytes
            archiveSha256 = [string] $built.result.archiveSha256
            portableFiles = [int] $inspected.result.portableFiles
            portableBytes = [int64] $inspected.result.portableBytes
            survivingIntermediateFiles = $unexpected.Count
            processTrace = $processTrace
        }
        environment = [ordered]@{
            os = [Environment]::OSVersion.VersionString
            architecture = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
            runnerImage = $env:ImageOS
            runnerImageVersion = $env:ImageVersion
            githubRunId = $env:GITHUB_RUN_ID
            githubRunAttempt = $env:GITHUB_RUN_ATTEMPT
            gitCommit = $env:GITHUB_SHA
        }
        measurement = [ordered]@{
            tool = 'scripts/measure-consumer-e2e.ps1'
            toolVersion = 1
            unit = 'milliseconds'
            endToEndClock = 'System.DateTime.UtcNow-across-workflow-steps'
            buildClock = 'System.Diagnostics.Stopwatch'
            startsAt = 'after-checkout-before-release-artifact-acquisition'
            endsAt = 'after-portable-zip-inspection'
            warmupSamples = 0
            concurrency = 1
            osFileCacheState = $osFileCacheState
        }
        error = $null
    }
    Write-Result -Value $result
    if ($AcquisitionMode -eq 'github-actions-artifact' -and $processTrace.status -ne 'pass') {
        $hostedGateFailed = $true
    }
} catch {
    $finishedAt = [DateTime]::UtcNow
    $failure = [ordered]@{
        schemaVersion = 'velox.consumer-e2e/v1'
        scope = 'checkout-complete-to-portable-zip'
        evidenceLevel = if ($AcquisitionMode -eq 'github-actions-artifact') { 'hosted-runner-evidence' } else { 'local-contract-smoke' }
        sampleId = $SampleId
        outcome = 'failure'
        startedAtUtc = $startedAt.ToString('o')
        finishedAtUtc = $finishedAt.ToString('o')
        durationMs = [Math]::Round(($finishedAt - $startedAt).TotalMilliseconds, 3)
        acquisition = $null
        release = $null
        fixture = $null
        build = $null
        environment = [ordered]@{
            os = [Environment]::OSVersion.VersionString
            architecture = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
            runnerImage = $env:ImageOS
            runnerImageVersion = $env:ImageVersion
            githubRunId = $env:GITHUB_RUN_ID
            githubRunAttempt = $env:GITHUB_RUN_ATTEMPT
            gitCommit = $env:GITHUB_SHA
        }
        measurement = [ordered]@{
            tool = 'scripts/measure-consumer-e2e.ps1'
            toolVersion = 1
            unit = 'milliseconds'
            endToEndClock = 'System.DateTime.UtcNow-across-workflow-steps'
            buildClock = 'System.Diagnostics.Stopwatch'
            startsAt = 'after-checkout-before-release-artifact-acquisition'
            endsAt = 'failure'
            warmupSamples = 0
            concurrency = 1
            osFileCacheState = $osFileCacheState
        }
        error = [ordered]@{
            phase = $phase
            code = 'PHASE_FAILED'
        }
    }
    try {
        Write-Result -Value $failure
    } catch {
        Write-Error 'Consumer end-to-end result could not be written or validated.'
    }
    throw
}

if ($hostedGateFailed) {
    throw 'Hosted child-process evidence did not pass.'
}
