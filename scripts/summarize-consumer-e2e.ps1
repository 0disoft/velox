[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string] $ResultsRoot,

    [Parameter(Mandatory = $true)]
    [string] $ResultPath,

    [Parameter(Mandatory = $true)]
    [ValidateRange(1, 100)]
    [int] $ExpectedSamples,

    [string] $RawSchemaPath = 'schema/consumer-e2e-v1.schema.json',

    [string] $SummarySchemaPath = 'schema/consumer-e2e-summary-v1.schema.json'
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

function Get-Percentile {
    param([double[]] $Values, [double] $Percentile)

    $sorted = @($Values | Sort-Object)
    $index = [Math]::Max(0, [Math]::Ceiling($Percentile * $sorted.Count) - 1)
    return [Math]::Round([double] $sorted[$index], 3)
}

$ResultsRoot = [System.IO.Path]::GetFullPath($ResultsRoot)
$ResultPath = [System.IO.Path]::GetFullPath($ResultPath)
$RawSchemaPath = [System.IO.Path]::GetFullPath($RawSchemaPath)
$SummarySchemaPath = [System.IO.Path]::GetFullPath($SummarySchemaPath)

foreach ($requiredPath in @($ResultsRoot, $RawSchemaPath, $SummarySchemaPath)) {
    if (-not (Test-Path -LiteralPath $requiredPath)) {
        throw "Required summary input is missing: $requiredPath"
    }
}

$resultFiles = if (Test-Path -LiteralPath $ResultsRoot -PathType Leaf) {
    @(Get-Item -LiteralPath $ResultsRoot)
} else {
    @(Get-ChildItem -LiteralPath $ResultsRoot -Recurse -File -Filter '*.json' | Sort-Object FullName)
}
$rawResults = [System.Collections.Generic.List[object]]::new()
foreach ($file in $resultFiles) {
    $json = [System.IO.File]::ReadAllText($file.FullName)
    if (-not ($json | Test-Json -SchemaFile $RawSchemaPath -ErrorAction Stop)) {
        throw "Raw consumer result does not satisfy its schema: $($file.Name)"
    }
    $rawResults.Add(($json | ConvertFrom-Json))
}

$sampleIds = @($rawResults | ForEach-Object { [string] $_.sampleId })
$duplicates = @($sampleIds | Group-Object | Where-Object Count -gt 1)
if ($duplicates.Count -gt 0) {
    throw 'Raw consumer results contain duplicate sample IDs.'
}

$successes = @($rawResults | Where-Object outcome -eq 'success')
$failures = @($rawResults | Where-Object outcome -eq 'failure')
$durations = [double[]] @($successes | ForEach-Object { [double] $_.durationMs })
$hostedOnly = $rawResults.Count -gt 0 -and @($rawResults | Where-Object evidenceLevel -ne 'hosted-runner-evidence').Count -eq 0
$releaseDigests = @($successes | ForEach-Object { [string] $_.acquisition.archiveSha256 } | Sort-Object -Unique)
$releaseDigest = if ($releaseDigests.Count -eq 1) { $releaseDigests[0] } else { $null }
$processPassCount = @($successes | Where-Object { $_.build.processTrace.status -eq 'pass' }).Count
$processFailCount = @($successes | Where-Object { $_.build.processTrace.status -eq 'fail' }).Count
$processUnverifiedCount = @($successes | Where-Object { $_.build.processTrace.status -eq 'unverified' }).Count
$statistics = if ($durations.Count -gt 0) {
    [ordered]@{
        minMs = [Math]::Round(($durations | Measure-Object -Minimum).Minimum, 3)
        p50Ms = Get-Percentile -Values $durations -Percentile 0.50
        p95Ms = Get-Percentile -Values $durations -Percentile 0.95
        maxMs = [Math]::Round(($durations | Measure-Object -Maximum).Maximum, 3)
    }
} else {
    $null
}

$result = [ordered]@{
    schemaVersion = 'velox.consumer-e2e-summary/v1'
    scope = 'checkout-complete-to-portable-zip'
    evidenceLevel = if ($hostedOnly) { 'hosted-runner-summary' } else { 'local-contract-summary' }
    expectedSamples = $ExpectedSamples
    observedSamples = $rawResults.Count
    successCount = $successes.Count
    failureCount = $failures.Count
    missingCount = [Math]::Max(0, $ExpectedSamples - $rawResults.Count)
    processEvidencePassCount = $processPassCount
    processEvidenceFailCount = $processFailCount
    processEvidenceUnverifiedCount = $processUnverifiedCount
    releaseArchiveSha256 = $releaseDigest
    statistics = $statistics
    samples = @($rawResults | Sort-Object sampleId | ForEach-Object {
        [ordered]@{
            sampleId = [string] $_.sampleId
            outcome = [string] $_.outcome
            durationMs = [double] $_.durationMs
            failurePhase = if ($_.error) { [string] $_.error.phase } else { $null }
            processEvidenceStatus = if ($_.build) { [string] $_.build.processTrace.status } else { $null }
        }
    })
}

$directory = [System.IO.Path]::GetDirectoryName($ResultPath)
New-Item -ItemType Directory -Path $directory -Force | Out-Null
$json = $result | ConvertTo-Json -Depth 8
if (-not ($json | Test-Json -SchemaFile $SummarySchemaPath -ErrorAction Stop)) {
    throw 'Consumer end-to-end summary does not satisfy its JSON schema.'
}
[System.IO.File]::WriteAllText($ResultPath, $json + [Environment]::NewLine, [System.Text.UTF8Encoding]::new($false))
$json

if ($result.observedSamples -ne $ExpectedSamples -or $result.failureCount -gt 0 -or $releaseDigests.Count -ne 1 -or ($hostedOnly -and ($processFailCount -gt 0 -or $processUnverifiedCount -gt 0))) {
    throw 'Consumer end-to-end evidence is incomplete or inconsistent.'
}
