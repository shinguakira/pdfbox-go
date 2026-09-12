<#
.SYNOPSIS
    Downloads the third-party PDF test suites the port is scored against, into
    go/testdata/corpus/.

.DESCRIPTION
    fetch-testdata.ps1 fetches what the Java build declares: reduced reproducers
    attached to numbered PDFBOX issues. That set is targeted at the bugs PDFBox
    has had, which is not the same as targeted at the format. It says nothing
    about the parts of ISO 32000 PDFBox has never been handed a bad file for.

    This script fetches the suites that are organised the other way round -- one
    file per clause, per feature, per construct -- so a gap in the port shows up
    as a named file rather than as a document that happens not to open.

    Each suite lands in its own directory under go/testdata/corpus/ and is
    listed by -List with what it covers. Nothing is committed: the whole
    directory is in go/.gitignore, and these carry other projects' licences.
    They are fetched, read as test input, and not redistributed.

    Sizes are the compressed download. `small` suites total well under 100 MB
    and are what -Suite defaults to; `large` ones are opt-in by name.

.PARAMETER Suite
    Which suites to fetch. Defaults to every suite marked small.

.PARAMETER List
    Print the suites and exit.

.PARAMETER Force
    Re-fetch a suite whose directory already exists.

.EXAMPLE
    pwsh go/migration/scripts/fetch-corpus.ps1 -List

.EXAMPLE
    pwsh go/migration/scripts/fetch-corpus.ps1

.EXAMPLE
    pwsh go/migration/scripts/fetch-corpus.ps1 -Suite pdfjs
#>
[CmdletBinding()]
param(
    [string[]]$Suite,

    [switch]$List,

    [switch]$Force,

    # Windows PowerShell leaves $PSScriptRoot empty while binding parameters.
    [string]$RepoRoot
)

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
[Net.ServicePointManager]::SecurityProtocol =
    [Net.SecurityProtocolType]::Tls12 -bor [Net.SecurityProtocolType]::Tls11

if (-not $RepoRoot) {
    $here = if ($PSScriptRoot) { $PSScriptRoot } else { Split-Path -Parent $MyInvocation.MyCommand.Path }
    $RepoRoot = (Resolve-Path (Join-Path $here '../../..')).Path
}

# ------------------------------------------------------------------ the suites

# Subtree is the path inside the archive to keep, '' for the whole thing. The
# archive's own top directory (repo-branch) is stripped either way.
$suites = @(
    [pscustomobject]@{
        Name    = 'pdf20examples'
        Size    = 'small'
        Bytes   = 100KB
        Licence = 'see LICENSE.md in the archive'
        Covers  = 'PDF 2.0 features one per file: UTF-8 strings, page-level output intents, black point compensation, 2.0-by-incremental-save, non-zero start offset'
        Exercises = 'cos, pdfparser, pdmodel/graphics/color, xmpbox'
        Url     = 'https://codeload.github.com/pdf-association/pdf20examples/tar.gz/refs/heads/master'
        Subtree = ''
    }
    [pscustomobject]@{
        Name    = 'safedocs-targeted'
        Size    = 'small'
        Bytes   = 2MB
        Licence = 'Apache-2.0'
        Covers  = 'hand-coded parser edge cases: dual startxref, Type 3 inside Type 3, page with no /Contents, fonts inside shading patterns, UTF-16LE strings, URI action trees'
        Exercises = 'pdfparser, contentstream, pdmodel/font, pdmodel/interactive/action'
        Url     = 'https://codeload.github.com/pdf-association/safedocs/tar.gz/refs/heads/main'
        Subtree = 'Miscellaneous Targeted Test PDFs'
    }
    [pscustomobject]@{
        Name    = 'verapdf'
        Size    = 'small'
        Bytes   = 74MB
        Licence = 'CC BY 4.0'
        Covers  = 'atomic ISO clause tests. One file asserts one requirement of ISO 19005 (PDF/A 1-4), ISO 14289 (PDF/UA 1-2), ISO 32000-1 and ISO 32000-2; the path and filename name the clause, and each file documents itself in its outline'
        Exercises = 'cos, pdfparser, xmpbox, pdmodel/documentinterchange, pdmodel/graphics/color'
        Url     = 'https://codeload.github.com/veraPDF/veraPDF-corpus/tar.gz/refs/heads/staging'
        Subtree = ''
    }
    [pscustomobject]@{
        Name    = 'qpdf'
        Size    = 'small'
        Bytes   = 44MB
        Licence = 'Apache-2.0'
        Covers  = 'cross-reference and object-stream permutations: linearized and not, object streams of every shape, damaged xref recovery, every encryption revision qpdf can write'
        Exercises = 'pdfparser, pdfwriter, cos, pdmodel/encryption'
        Url     = 'https://codeload.github.com/qpdf/qpdf/tar.gz/refs/heads/main'
        Subtree = 'qpdf/qtest/qpdf'
    }
    [pscustomobject]@{
        Name    = 'text-rendering-tests'
        Size    = 'small'
        Bytes   = 9MB
        Licence = 'see the archive; fonts are OFL'
        Covers  = 'shaping ground truth independent of PDFBox: test fonts with expected glyph ids and positions per string, for GSUB and GPOS features'
        Exercises = 'pdfbox/glyphlayout and fontbox/ttf gsub/gpos -- the one part of the port written from a specification rather than ported, so the only part with no Java to check against'
        Url     = 'https://codeload.github.com/unicode-org/text-rendering-tests/tar.gz/refs/heads/main'
        Subtree = ''
    }
    [pscustomobject]@{
        Name    = 'pdfjs'
        Size    = 'large'
        Bytes   = 195MB
        Licence = 'Apache-2.0'
        Covers  = "reduced reproducers from another reader's bug tracker. Not ground truth -- pdf.js's expectations are pdf.js's -- but 982 committed files that broke a real implementation, which is what makes them worth opening; the other 459 entries are .link stubs pdf.js fetches on demand and this does not"
        Exercises = 'everything; use as crash and regression input, not as assertions'
        Url     = 'https://codeload.github.com/mozilla/pdf.js/tar.gz/refs/heads/master'
        Subtree = 'test/pdfs'
    }
)

# pdfCabinetOfHorrors is a handful of files inside a 520 MB repository, so it is
# fetched file by file through the contents API instead of by archive.
$apiSuites = @(
    [pscustomobject]@{
        Name    = 'cabinet-of-horrors'
        Size    = 'small'
        Bytes   = 3MB
        Licence = 'see the format-corpus repository'
        Covers  = 'the files digital preservation keeps tripping over: broken embedded fonts, encrypted-without-password, malformed page trees'
        Exercises = 'pdfparser, pdmodel/font, pdmodel/encryption'
        Api     = 'https://api.github.com/repos/openpreserve/format-corpus/contents/pdfCabinetOfHorrors'
    }
)

$all = @($suites) + @($apiSuites)

if ($List) {
    foreach ($s in $all) {
        ''
        "{0}  [{1}, ~{2:N0} MB]" -f $s.Name, $s.Size, ($s.Bytes / 1MB)
        "  licence   : $($s.Licence)"
        "  covers    : $($s.Covers)"
        "  exercises : $($s.Exercises)"
    }
    ''
    "$($all.Count) suites. Default is every suite marked small."
    return
}

if (-not $Suite) {
    $Suite = ($all | Where-Object { $_.Size -eq 'small' }).Name
}

$unknown = $Suite | Where-Object { $all.Name -notcontains $_ }
if ($unknown) { throw "unknown suite: $($unknown -join ', '). Run with -List." }

$corpusRoot = Join-Path $RepoRoot 'go/testdata/corpus'
New-Item -ItemType Directory -Force -Path $corpusRoot | Out-Null

# ------------------------------------------------------------------ the fetch

# bsdtar ships with Windows 10 1803 and later, and with every platform pwsh runs
# on. It is used rather than Expand-Archive because these are .tar.gz.
if (-not (Get-Command tar -ErrorAction SilentlyContinue)) {
    throw 'tar is not on PATH; it is needed to unpack the suite archives'
}

function Get-ArchiveSuite {
    param([object]$S, [string]$Dest)

    $tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("corpus-" + [guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Force -Path $tmp | Out-Null
    try {
        $archive = Join-Path $tmp 'suite.tar.gz'
        Invoke-WebRequest -Uri $S.Url -OutFile $archive -TimeoutSec 900 -UseBasicParsing

        $extract = Join-Path $tmp 'x'
        New-Item -ItemType Directory -Force -Path $extract | Out-Null
        # --strip-components=1 drops the repo-branch directory GitHub wraps
        # every source archive in.
        & tar -xzf $archive -C $extract --strip-components=1
        if ($LASTEXITCODE -ne 0) { throw "tar exited $LASTEXITCODE" }

        $source = if ($S.Subtree) { Join-Path $extract $S.Subtree } else { $extract }
        if (-not (Test-Path -LiteralPath $source)) {
            throw "subtree '$($S.Subtree)' is not in the archive -- has the repository moved it?"
        }

        if (Test-Path -LiteralPath $Dest) { Remove-Item -LiteralPath $Dest -Recurse -Force }
        New-Item -ItemType Directory -Force -Path (Split-Path -Parent $Dest) | Out-Null
        Move-Item -LiteralPath $source -Destination $Dest
    }
    finally {
        Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
    }
}

function Get-ApiSuite {
    param([object]$S, [string]$Dest)

    $headers = @{ 'User-Agent' = 'pdfbox-go-fetch-corpus' }
    $entries = Invoke-RestMethod -Uri $S.Api -Headers $headers -TimeoutSec 120

    New-Item -ItemType Directory -Force -Path $Dest | Out-Null
    foreach ($e in $entries) {
        if ($e.type -ne 'file') { continue }
        Invoke-WebRequest -Uri $e.download_url -OutFile (Join-Path $Dest $e.name) `
            -TimeoutSec 300 -UseBasicParsing
    }
}

foreach ($name in $Suite) {
    $s = $all | Where-Object { $_.Name -eq $name } | Select-Object -First 1
    $dest = Join-Path $corpusRoot $name

    if ((Test-Path -LiteralPath $dest) -and -not $Force) {
        $have = (Get-ChildItem -LiteralPath $dest -Recurse -File -Filter *.pdf -ErrorAction SilentlyContinue).Count
        "$name -- already present, $have PDFs (pass -Force to refetch)"
        continue
    }

    "$name -- fetching ~$([math]::Round($s.Bytes / 1MB)) MB ..."
    if ($s.PSObject.Properties.Name -contains 'Api') {
        Get-ApiSuite -S $s -Dest $dest
    }
    else {
        Get-ArchiveSuite -S $s -Dest $dest
    }

    $pdfs = (Get-ChildItem -LiteralPath $dest -Recurse -File -Filter *.pdf -ErrorAction SilentlyContinue).Count
    $bytes = (Get-ChildItem -LiteralPath $dest -Recurse -File | Measure-Object -Property Length -Sum).Sum
    "$name -- $pdfs PDFs, $([math]::Round($bytes / 1MB, 1)) MB on disk"
}

''
'corpus root: ' + $corpusRoot
Get-ChildItem -LiteralPath $corpusRoot -Directory | ForEach-Object {
    $n = (Get-ChildItem -LiteralPath $_.FullName -Recurse -File -Filter *.pdf -ErrorAction SilentlyContinue).Count
    "  {0,-22} {1,6} PDFs" -f $_.Name, $n
}
