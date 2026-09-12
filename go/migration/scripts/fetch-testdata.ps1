<#
.SYNOPSIS
    Downloads the test files the Java build declares, into the directories the
    Java build puts them in.

.DESCRIPTION
    pdfbox/pom.xml, fontbox/pom.xml, examples/pom.xml and benchmark/pom.xml each
    configure download-maven-plugin executions that fetch test PDFs, fonts and
    images into <module>/target/pdfs, /fonts and /imgs. PDFBOX-3974 added them so
    that files too large or too licence-encumbered to commit could still be test
    input; every one is pinned by a SHA-512 the pom carries, and every one is the
    reduced reproducer attached to a numbered PDFBOX issue.

    The port's tests already read those directories and skip when they are empty,
    which is every checkout, because nothing here runs Maven. Roughly thirty Go
    test cases are written but never run for that reason, and they are the ones
    that carry a real document rather than a hand-built dictionary. This script
    fills the directories without needing a JDK.

    The list is read out of the poms rather than copied into this file, so it
    cannot drift from the frozen Java tree. A file already present and matching
    its SHA-512 is left alone, so re-running costs one hash per file.

    Nothing downloaded here is committed: target/ is in the repository
    .gitignore, and these are third-party documents attached to public issues.
    They are fetched, used as test input, and never redistributed.

.PARAMETER RepoRoot
    Repository root. Defaults to three levels above this script.

.PARAMETER Module
    Which poms to read. Defaults to all four that declare downloads.

.PARAMETER Kind
    Restrict to one output directory: pdfs, fonts or imgs.

.PARAMETER List
    Print the manifest and exit without downloading.

.PARAMETER Force
    Re-download even when the file is present and verifies.

.EXAMPLE
    pwsh go/migration/scripts/fetch-testdata.ps1 -List

.EXAMPLE
    pwsh go/migration/scripts/fetch-testdata.ps1

.EXAMPLE
    pwsh go/migration/scripts/fetch-testdata.ps1 -Module pdfbox -Kind pdfs
#>
[CmdletBinding()]
param(
    # Windows PowerShell leaves $PSScriptRoot empty while binding parameters,
    # so this is resolved in the body rather than here.
    [string]$RepoRoot,

    [ValidateSet('pdfbox', 'fontbox', 'examples', 'benchmark', 'parent')]
    [string[]]$Module = @('pdfbox', 'fontbox', 'examples', 'benchmark'),

    [ValidateSet('pdfs', 'fonts', 'imgs')]
    [string[]]$Kind,

    [switch]$List,

    [switch]$Force,

    [int]$Retry = 3
)

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

if (-not $RepoRoot) {
    $here = if ($PSScriptRoot) { $PSScriptRoot } else { Split-Path -Parent $MyInvocation.MyCommand.Path }
    $RepoRoot = (Resolve-Path (Join-Path $here '../../..')).Path
}

[Net.ServicePointManager]::SecurityProtocol =
    [Net.SecurityProtocolType]::Tls12 -bor [Net.SecurityProtocolType]::Tls11

# ---------------------------------------------------------------- the manifest

# Read the download-maven-plugin executions out of a pom. Each one gives a URL,
# the name to save it under, the directory to put it in, and the SHA-512 the
# Java build checks it against.
function Read-PomDownloads {
    param([string]$PomPath, [string]$ModuleName)

    if (-not (Test-Path -LiteralPath $PomPath)) { return }

    [xml]$pom = Get-Content -LiteralPath $PomPath
    $ns = New-Object System.Xml.XmlNamespaceManager($pom.NameTable)
    $ns.AddNamespace('m', 'http://maven.apache.org/POM/4.0.0')

    $query = "//m:plugin[m:artifactId='download-maven-plugin']//m:execution"
    foreach ($exec in $pom.SelectNodes($query, $ns)) {
        $cfg = $exec.SelectSingleNode('m:configuration', $ns)
        if ($null -eq $cfg) { continue }

        $urlNode = $cfg.SelectSingleNode('m:url', $ns)
        if ($null -eq $urlNode) { continue }
        $url = $urlNode.InnerText.Trim()

        $idNode = $exec.SelectSingleNode('m:id', $ns)
        $nameNode = $cfg.SelectSingleNode('m:outputFileName', $ns)
        $dirNode = $cfg.SelectSingleNode('m:outputDirectory', $ns)
        $shaNode = $cfg.SelectSingleNode('m:sha512', $ns)
        $unpackNode = $cfg.SelectSingleNode('m:unpack', $ns)

        # No outputFileName means the plugin keeps the name from the URL, which
        # is percent-encoded in a fair number of these.
        if ($null -ne $nameNode) {
            $name = $nameNode.InnerText.Trim()
        }
        else {
            $name = [System.Uri]::UnescapeDataString(($url -split '/')[-1])
        }

        $dir = if ($null -ne $dirNode) { $dirNode.InnerText.Trim() } else { '' }
        $dir = $dir.Replace('${project.build.directory}', 'target')

        [pscustomobject]@{
            Module = $ModuleName
            Id     = if ($null -ne $idNode) { $idNode.InnerText.Trim() } else { '' }
            Name   = $name
            Dir    = $dir
            Kind   = ($dir -split '[/\\]')[-1]
            Sha512 = if ($null -ne $shaNode) { $shaNode.InnerText.Trim().ToLowerInvariant() } else { '' }
            Unpack = ($null -ne $unpackNode -and $unpackNode.InnerText.Trim() -eq 'true')
            Url    = $url
        }
    }
}

$manifest = [System.Collections.Generic.List[object]]::new()
foreach ($m in $Module) {
    Read-PomDownloads -PomPath (Join-Path $RepoRoot "$m/pom.xml") -ModuleName $m |
        ForEach-Object { $manifest.Add($_) }
}

if ($Kind) {
    $manifest = [System.Collections.Generic.List[object]](
        $manifest | Where-Object { $Kind -contains $_.Kind })
}

if ($manifest.Count -eq 0) {
    Write-Warning 'no download executions found -- has the pom layout changed?'
    return
}

if ($List) {
    $manifest |
        Sort-Object Module, Kind, Name |
        Format-Table -AutoSize Module, Kind, Name, Unpack, Url
    "$($manifest.Count) files"
    return
}

# ------------------------------------------------------------------ the fetch

function Get-Sha512 {
    param([string]$Path)
    (Get-FileHash -LiteralPath $Path -Algorithm SHA512).Hash.ToLowerInvariant()
}

# True when the file is already there and is the file the pom asked for. An
# unpack entry is judged by the archive kept beside the extraction, because the
# extracted tree has no hash of its own.
function Test-AlreadyFetched {
    param([object]$Entry, [string]$Path)

    if (-not (Test-Path -LiteralPath $Path)) { return $false }
    if ($Entry.Sha512 -eq '') { return $true }
    return (Get-Sha512 $Path) -eq $Entry.Sha512
}

$fetched = 0
$kept = 0
$failed = [System.Collections.Generic.List[string]]::new()
$bytes = 0L
$index = 0

# The tests come before the benchmark inputs, which are large, hosted off the
# Apache infrastructure, and wanted by nothing under go/.
$order = @{ pdfbox = 0; fontbox = 1; examples = 2; parent = 3; benchmark = 4 }

foreach ($entry in ($manifest | Sort-Object @{ Expression = { $order[$_.Module] } }, Kind, Name)) {
    $index++
    $outDir = Join-Path $RepoRoot (Join-Path $entry.Module $entry.Dir)

    # An unpack entry extracts into the output directory; the archive itself is
    # parked under .archives so a re-run can verify it without re-downloading.
    if ($entry.Unpack) {
        $archiveDir = Join-Path $outDir '.archives'
        $target = Join-Path $archiveDir $entry.Name
    }
    else {
        $target = Join-Path $outDir $entry.Name
    }

    $label = '{0,3}/{1} {2}' -f $index, $manifest.Count, $entry.Name

    if (-not $Force -and (Test-AlreadyFetched -Entry $entry -Path $target)) {
        $kept++
        Write-Verbose "$label -- already present"
        continue
    }

    New-Item -ItemType Directory -Force -Path (Split-Path -Parent $target) | Out-Null

    $ok = $false
    for ($attempt = 1; $attempt -le $Retry -and -not $ok; $attempt++) {
        $temp = "$target.part"
        try {
            Invoke-WebRequest -Uri $entry.Url -OutFile $temp -TimeoutSec 180 -UseBasicParsing

            if ($entry.Sha512 -ne '') {
                $got = Get-Sha512 $temp
                if ($got -ne $entry.Sha512) {
                    # Deterministic: the host is serving something other than
                    # what the pom pinned, and fetching it again will not
                    # change that. Give up on this entry rather than burn the
                    # retries -- it is usually a moved file or an error page.
                    $attempt = $Retry
                    throw "SHA-512 mismatch: pom says $($entry.Sha512.Substring(0,16))..., got $($got.Substring(0,16))..."
                }
            }

            $size = (Get-Item -LiteralPath $temp).Length
            Move-Item -LiteralPath $temp -Destination $target -Force
            $bytes += $size
            $ok = $true

            if ($entry.Unpack) {
                Expand-Archive -LiteralPath $target -DestinationPath $outDir -Force
                "$label  $([math]::Round($size/1KB)) KB, unpacked"
            }
            else {
                "$label  $([math]::Round($size/1KB)) KB"
            }
            $fetched++
        }
        catch {
            if (Test-Path -LiteralPath $temp) { Remove-Item -LiteralPath $temp -Force }
            # A host that does not resolve will not resolve on the next try
            # either; two of the benchmark inputs are hosted somewhere that has
            # since gone away.
            if ($_.Exception.Message -match 'could not be resolved|No such host') { $attempt = $Retry }
            if ($attempt -eq $Retry) {
                $failed.Add("$($entry.Name) <- $($entry.Url) : $($_.Exception.Message)")
                Write-Warning "$label -- $($_.Exception.Message)"
            }
            else {
                Start-Sleep -Seconds (2 * $attempt)
            }
        }
    }
}

''
"fetched $fetched, already present $kept, failed $($failed.Count) -- $([math]::Round($bytes/1MB,1)) MB downloaded"

if ($failed.Count -gt 0) {
    ''
    'failures:'
    $failed | ForEach-Object { "  $_" }
    exit 1
}
