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

    Sizes are the compressed download, or for a suite fetched with git the size
    of the files kept. `small` suites total well under 100 MB and are what
    -Suite defaults to; `large` ones are opt-in by name. The two iText suites
    need git on PATH.

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

.EXAMPLE
    pwsh go/migration/scripts/fetch-corpus.ps1 -Suite itext-java,itext-dotnet
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
        Covers  = "every PDF pdf.js tests with: the 982 committed in test/pdfs; the 459 it keeps out of its repository, each named by a <file>.link stub holding a URL, downloaded here and checked against the md5 in test/test_manifest.json as pdf.js's own runner checks them; and the two PDFs committed outside test/pdfs, under _repo/. The passwords pdf.js opens its 12 encrypted files with are written to _passwords.tsv. Not on disk: test/pdfs/sig_corpus, eight signed PDFs pdf.js does not commit or download, which its generate.py builds with a mozilla-central checkout. Not ground truth -- pdf.js's expectations are pdf.js's -- but files that broke a real implementation"
        Exercises = 'everything; use as crash and regression input, not as assertions'
        Url     = 'https://codeload.github.com/mozilla/pdf.js/tar.gz/refs/heads/master'
        Subtree = 'test/pdfs'
        # Copied from the archive into _repo/, keeping their paths: the PDFs pdf.js
        # commits outside test/pdfs, and the manifest the linked files are checked
        # against.
        Extras  = @('web/compressed.tracemonkey-pldi-09.pdf', 'examples/learning/helloworld.pdf', 'test/test_manifest.json')
        # Resolve the .link stubs, checking each download against this manifest.
        LinkManifest = '_repo/test/test_manifest.json'
        # The passwords pdf.js opens its encrypted files with. The manifest gives
        # some, as "password" on the file's entry, and is read for them; these
        # are the rest, which only its test code gives. Written, both together,
        # to _passwords.tsv for run-oracle.ps1 -Passwords and cmd/corpus -passwords.
        PasswordManifest = '_repo/test/test_manifest.json'
        Passwords = [ordered]@{
            'pr6531_1.pdf'             = 'asdfasdf' # test/unit/api_spec.js
            'pr6531_2.pdf'             = 'asdfasdf' # test/unit/api_spec.js
            'auth-event-ef-open.pdf'   = '000000'   # test/unit/api_spec.js
            'encrypted-attachment.pdf' = '000000'   # test/unit/api_spec.js
            'print_protection.pdf'     = '1234'     # test/integration/viewer_spec.mjs
        }
    }
)

# iText's PDFs are 542 MB of a 1.4 GB tree in each of its two repositories, spread
# through the test resources of every module, so they are fetched by a partial
# clone and a sparse checkout of the Keep patterns rather than by archive. Each
# file keeps its path in the repository.
$gitSuites = @(
    [pscustomobject]@{
        Name    = 'itext-java'
        Size    = 'large'
        Bytes   = 542MB
        Licence = 'AGPL-3.0, or commercial from Apryse'
        Covers  = 'every PDF committed to iText Core for Java, at its path in the repository: 6,897 on develop at 2026-09-14, all in test resources -- layout 2,342, kernel 1,779, svg 1,446, forms 682, sign 386, pdfa 172, barcodes 39, brotli-compressor 20, pdfua 18, webp-image-support 8, pdftest 3, io 2. The inputs its tests read and the cmp_ files they compare their output with; nothing its tests read comes from the network. 234 name an encryption dictionary, and passwords/itext-java.tsv gives the ways its tests open them -- the passwords, and for the 40 encrypted for a certificate the certificate and key, which are fetched too -- written to _passwords.tsv. Not on disk: the PDFs the tests write, which exist only once the Maven build has run them'
        Exercises = 'everything; kernel and forms reach cos, pdfparser, pdmodel/encryption and pdmodel/interactive/form most directly'
        Git     = 'https://github.com/itext/itext-java.git'
        Branch  = 'develop'
        Keep    = @('*.[pP][dD][fF]')
        PasswordTable = 'passwords/itext-java.tsv'
    }
    [pscustomobject]@{
        Name    = 'itext-dotnet'
        Size    = 'large'
        Bytes   = 545MB
        Licence = 'AGPL-3.0, or commercial from Apryse'
        Covers  = 'every PDF committed to iText Core for .NET, at its path in the repository: 6,960 on develop at 2026-09-14, all under itext.tests. The same library ported from the Java, and mostly the same files: 6,678 of its 6,780 distinct contents are in itext-java too, and the other 102 -- 80 of them in itext.sign.tests -- are not. 240 name an encryption dictionary; passwords/itext-dotnet.tsv, as for itext-java'
        Exercises = 'as itext-java'
        Git     = 'https://github.com/itext/itext-dotnet.git'
        Branch  = 'develop'
        Keep    = @('*.[pP][dD][fF]')
        PasswordTable = 'passwords/itext-dotnet.tsv'
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

$all = @($suites) + @($gitSuites) + @($apiSuites)

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

        if ($S.PSObject.Properties.Name -contains 'Extras') {
            foreach ($extra in $S.Extras) {
                $from = Join-Path $extract $extra
                if (-not (Test-Path -LiteralPath $from)) {
                    throw "'$extra' is not in the archive -- has the repository moved it?"
                }
                $to = Join-Path (Join-Path $Dest '_repo') $extra
                New-Item -ItemType Directory -Force -Path (Split-Path -Parent $to) | Out-Null
                Copy-Item -LiteralPath $from -Destination $to
            }
        }
    }
    finally {
        Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
    }
}

# Invoke-CurlRound downloads, eight at a time, one candidate URL for each item
# still missing, and keeps a download only if its md5 is the one the manifest
# records. What a round could not settle is left for the next, with the attempt
# written down. curl.exe is used rather than Invoke-WebRequest for the
# parallelism and the retries; it ships with Windows 10 1803 and later, as tar
# does.
function Invoke-CurlRound {
    param([object[]]$Items, [scriptblock]$CandidateOf)

    $round = @()
    foreach ($item in $Items) {
        if ($item.Done) { continue }
        $candidate = & $CandidateOf $item
        if (-not $candidate -or $item.Tried.Contains($candidate)) { continue }
        [void]$item.Tried.Add($candidate)
        $item.Current = $candidate
        $round += $item
    }
    if ($round.Count -eq 0) { return }

    $config = Join-Path ([System.IO.Path]::GetTempPath()) ("links-" + [guid]::NewGuid().ToString('N') + ".txt")
    $lines = foreach ($item in $round) {
        'url = "' + ($item.Current -replace '\\', '\\' -replace '"', '\"') + '"'
        'output = "' + ("$($item.Target).part" -replace '\\', '\\' -replace '"', '\"') + '"'
    }
    [System.IO.File]::WriteAllLines($config, [string[]]$lines, (New-Object System.Text.UTF8Encoding $false))
    try {
        & curl.exe --parallel --parallel-max 8 --location --max-redirs 20 --retry 3 --retry-delay 2 `
            --connect-timeout 60 --max-time 1800 --fail --silent --show-error `
            --user-agent 'pdfbox-go-fetch-corpus' --config $config 2>$null
    }
    finally {
        Remove-Item -LiteralPath $config -Force -ErrorAction SilentlyContinue
    }

    foreach ($item in $round) {
        $part = "$($item.Target).part"
        if (-not (Test-Path -LiteralPath $part)) {
            $item.Attempts.Add("$($item.Current) -> nothing arrived")
            continue
        }
        $hash = (Get-FileHash -Algorithm MD5 -LiteralPath $part).Hash.ToLowerInvariant()
        if ($hash -eq $item.Md5) {
            Move-Item -Force -LiteralPath $part -Destination $item.Target
            $item.Done = $true
            $item.Source = $item.Current
        }
        else {
            $item.Attempts.Add("$($item.Current) -> md5 $hash")
            Remove-Item -LiteralPath $part -Force -ErrorAction SilentlyContinue
        }
    }
}

# Get-Captures asks archive.org's capture index for every capture of a URL that
# answered 200, as raw-bytes URLs. The index is slow and often busy, so it is
# only asked about files the link itself could not supply, and asked again a few
# times before its silence is taken as an answer.
function Get-Captures {
    param([string]$Url)

    $original = $Url -replace '^https?://web\.archive\.org/web/\d+(?:id_)?/', ''
    $query = "https://web.archive.org/cdx/search/cdx?output=json&filter=statuscode:200&url=" + [uri]::EscapeDataString($original)
    for ($try = 1; $try -le 4; $try++) {
        $json = & curl.exe --location --silent --fail --max-time 180 --user-agent 'pdfbox-go-fetch-corpus' $query 2>$null
        if ($LASTEXITCODE -eq 0 -and $json) {
            $rows = @(($json -join "`n") | ConvertFrom-Json)
            return @($rows | Select-Object -Skip 1 | ForEach-Object { "https://web.archive.org/web/$($_[1])id_/$($_[2])" })
        }
        Start-Sleep -Seconds (10 * $try)
    }
    return @()
}

# Resolve-Links does for a suite what pdf.js's test runner does before it runs:
# every <file>.link stub names a PDF the project keeps out of its repository, and
# each is downloaded and checked against the md5 its manifest records.
#
# Three rounds, each only for what the one before could not settle: the link as
# written; for an archive.org link, the same capture asked for its raw bytes; and
# then every capture archive.org holds of the file's original address. A file
# that still has not arrived with the right checksum is named in _links.tsv with
# everything that was tried, and the run fails -- a partial suite must not look
# whole. Files already present with the right checksum are not fetched again, so
# a run that stopped part way resumes where it stopped.
function Resolve-Links {
    param([object]$S, [string]$Dest)

    if (-not (Get-Command curl.exe -ErrorAction SilentlyContinue)) {
        throw 'curl.exe is not on PATH; it is needed to fetch the linked files'
    }

    $manifestPath = Join-Path $Dest $S.LinkManifest
    $expected = @{}
    foreach ($entry in (Get-Content -Raw -Encoding UTF8 -LiteralPath $manifestPath | ConvertFrom-Json)) {
        if ($entry.md5) {
            $expected[($entry.file -replace '^pdfs/', '')] = $entry.md5.ToLowerInvariant()
        }
    }

    # Where each file came from, as the run that fetched it wrote down, so that
    # a later run that finds the file already present does not lose it.
    $reportPath = Join-Path $Dest '_links.tsv'
    $previous = @{}
    if (Test-Path -LiteralPath $reportPath) {
        foreach ($row in (Get-Content -Encoding UTF8 -LiteralPath $reportPath | Select-Object -Skip 1)) {
            $fields = $row -split "`t", 3
            if ($fields.Count -eq 3 -and $fields[1] -eq 'ok' -and $fields[2] -ne 'already present') {
                $previous[$fields[0]] = $fields[2]
            }
        }
    }

    $items = @()
    foreach ($link in @(Get-ChildItem -LiteralPath $Dest -Recurse -File -Filter '*.link')) {
        $name = $link.Name.Substring(0, $link.Name.Length - '.link'.Length)
        $target = Join-Path $link.DirectoryName $name
        $item = [pscustomobject]@{
            Name     = $name
            Target   = $target
            Md5      = $expected[$name]
            Url      = (Get-Content -LiteralPath $link.FullName | Where-Object { $_.Trim() } | Select-Object -First 1).Trim()
            Done     = $false
            Source   = ''
            Current  = ''
            Tried    = New-Object System.Collections.Generic.HashSet[string]
            Attempts = New-Object System.Collections.Generic.List[string]
        }
        if (-not $item.Md5) {
            $item.Attempts.Add('no md5 for it in the manifest')
        }
        elseif ((Test-Path -LiteralPath $target) -and
            (Get-FileHash -Algorithm MD5 -LiteralPath $target).Hash.ToLowerInvariant() -eq $item.Md5) {
            $item.Done = $true
            $item.Source = if ($previous.ContainsKey($name)) { $previous[$name] } else { 'already present' }
        }
        $items += $item
    }
    $withMd5 = @($items | Where-Object { $_.Md5 })

    Write-Host "  links: $($items.Count), already present $(@($items | Where-Object { $_.Done }).Count); round 1, the links as written ..."
    Invoke-CurlRound -Items $withMd5 -CandidateOf { param($i) $i.Url }

    Write-Host "  links: $(@($items | Where-Object { -not $_.Done }).Count) left; round 2, archive.org captures asked for raw bytes ..."
    Invoke-CurlRound -Items $withMd5 -CandidateOf {
        param($i)
        if ($i.Url -match '^https?://web\.archive\.org/web/(\d+)/(.+)$') { "https://web.archive.org/web/$($Matches[1])id_/$($Matches[2])" }
    }

    $left = @($withMd5 | Where-Object { -not $_.Done })
    Write-Host "  links: $($left.Count) left; round 3, every capture archive.org holds ..."
    foreach ($item in $left) {
        $captures = @(Get-Captures -Url $item.Url)
        if ($captures.Count -eq 0) {
            $item.Attempts.Add('archive.org capture index: no capture, or no answer')
        }
        foreach ($capture in $captures) {
            Invoke-CurlRound -Items @($item) -CandidateOf { param($i) $capture }.GetNewClosure()
            if ($item.Done) { break }
        }
    }

    $report = New-Object System.Collections.Generic.List[string]
    $report.Add("file`tresult`tsource, or everything tried")
    $failed = 0
    foreach ($item in ($items | Sort-Object Name)) {
        if ($item.Done) {
            $report.Add("$($item.Name)`tok`t$($item.Source)")
        }
        else {
            $failed++
            $report.Add("$($item.Name)`tMISSING`t$($item.Attempts -join ' | ')")
        }
    }
    [System.IO.File]::WriteAllLines($reportPath, $report, (New-Object System.Text.UTF8Encoding $false))
    Write-Host "$($S.Name) -- $($items.Count) linked files: $($items.Count - $failed) present with the manifest's md5, $failed missing; see $reportPath"
    return $failed
}

# Write-Passwords writes _passwords.tsv for a suite whose project publishes the
# passwords of its encrypted files, in UTF-8 without a byte order mark: one line
# for each way of opening a file, the file's path under the corpus root, a tab,
# and the password -- or, for a file encrypted for a certificate, the password,
# the certificate and the private key, tab separated, those two relative to the
# suite. Both drivers match a line against the end of the path they were given,
# so the table reads the same whether they run from the repository root or from
# go/.
function Write-Passwords {
    param([object]$S, [string]$Dest)

    $table = [ordered]@{}
    function Add-Row([string]$File, [string]$Password, [string]$From) {
        if ($table.Contains($File) -and $table[$File] -ne $Password) {
            throw "$($S.Name): two passwords for $File, the second from $From"
        }
        $table[$File] = $Password
    }

    if ($S.PSObject.Properties.Name -contains 'PasswordManifest') {
        $manifestPath = Join-Path $Dest $S.PasswordManifest
        foreach ($entry in (Get-Content -Raw -Encoding UTF8 -LiteralPath $manifestPath | ConvertFrom-Json)) {
            if ($entry.PSObject.Properties.Name -contains 'password') {
                Add-Row ($entry.file -replace '^pdfs/', '') $entry.password $S.PasswordManifest
            }
        }
    }
    if ($S.PSObject.Properties.Name -contains 'Passwords') {
        foreach ($file in $S.Passwords.Keys) {
            Add-Row $file $S.Passwords[$file] 'the suite definition'
        }
    }

    $lines = New-Object System.Collections.Generic.List[string]
    foreach ($file in $table.Keys) {
        if (-not (Test-Path -LiteralPath (Join-Path $Dest $file))) {
            throw "$($S.Name): a password is recorded for $file, which is not in the suite"
        }
        $lines.Add("$($S.Name)/$file`t$($table[$file])")
    }

    # A suite whose tests open files more than one way, or with certificates,
    # keeps its table in passwords/ beside this script, one line for each way of
    # opening a file, paths relative to the suite. The lines are copied with the
    # suite's name in front of each file, and every file a line names -- the PDF,
    # and the certificate and key of a certificate line -- has to be there.
    $opened = @{}
    if ($S.PSObject.Properties.Name -contains 'PasswordTable') {
        foreach ($line in (Read-PasswordTable -S $S)) {
            if ($line.StartsWith('#') -or -not $line.Trim()) {
                $lines.Add($line)
                continue
            }
            $fields = $line -split "`t"
            foreach ($named in @($fields[0]) + @($fields | Select-Object -Skip 2)) {
                if (-not (Test-Path -LiteralPath (Join-Path $Dest $named))) {
                    throw "$($S.Name): the passwords table names $named, which is not in the suite -- if the suite was fetched before the table named it, fetch it again with -Force"
                }
            }
            $opened[$fields[0]] = $true
            $lines.Add("$($S.Name)/$line")
        }
    }

    $path = Join-Path $Dest '_passwords.tsv'
    [System.IO.File]::WriteAllLines($path, $lines, (New-Object System.Text.UTF8Encoding $false))
    Write-Host "$($S.Name) -- passwords for $($table.Count + $opened.Count) encrypted files; see $path"
}

# Read-PasswordTable answers the lines of a suite's committed passwords table.
function Read-PasswordTable {
    param([object]$S)

    $path = Join-Path (Join-Path $RepoRoot 'go/migration/scripts') $S.PasswordTable
    $lines = [System.IO.File]::ReadAllLines($path, (New-Object System.Text.UTF8Encoding $false))
    foreach ($line in $lines) {
        if ($line.StartsWith('#') -or -not $line.Trim()) { continue }
        $count = ($line -split "`t").Count
        if ($count -ne 2 -and $count -ne 4) {
            throw "$path`: a line that is neither a file and a password nor a file, a password, a certificate and a key: $line"
        }
    }
    return $lines
}

# Get-TableKeys answers the certificates and keys a suite's passwords table
# names, which the sparse checkout has to bring as well as the PDFs.
function Get-TableKeys {
    param([object]$S)

    if ($S.PSObject.Properties.Name -notcontains 'PasswordTable') { return @() }
    $keys = [ordered]@{}
    foreach ($line in (Read-PasswordTable -S $S)) {
        if ($line.StartsWith('#') -or -not $line.Trim()) { continue }
        $fields = $line -split "`t"
        foreach ($named in ($fields | Select-Object -Skip 2)) { $keys[$named] = $true }
    }
    return @($keys.Keys)
}

# Get-GitSuite fetches a suite whose files are scattered through a repository
# much larger than they are: iText keeps its PDFs in the test resources of a
# dozen modules, beside more than their size again in fonts, images and
# sources. A partial clone brings the tree without the contents, a sparse
# checkout of the suite's Keep patterns brings the contents of those files
# alone, and each file keeps its path in the repository.
#
# Nothing is converted on the way out, whatever the repository's attributes say
# about line endings, and every file is then checked against the blob id the
# repository records for it: a file that is not byte for byte what is committed,
# or a committed file that did not arrive, fails the run. _revision.txt records
# the commit the files are from.
function Get-GitSuite {
    param([object]$S, [string]$Dest)

    if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
        throw 'git is not on PATH; it is needed to fetch this suite'
    }

    # Staged and moved into place, as the archive suites are.
    $tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("corpus-" + [guid]::NewGuid().ToString('N'))
    $config = @('-c', 'core.autocrlf=false', '-c', 'core.eol=lf', '-c', 'core.longpaths=true', '-c', 'core.quotepath=false')
    # git writes paths in UTF-8, and PowerShell reads a native program's output
    # in the console's code page unless told otherwise.
    $savedConsole = [Console]::OutputEncoding
    [Console]::OutputEncoding = New-Object System.Text.UTF8Encoding $false
    try {
        # The Keep patterns, and each certificate and key the suite's passwords
        # table names, anchored to the root.
        $patterns = @($S.Keep) + @(Get-TableKeys -S $S | ForEach-Object { "/$_" })
        & git @config clone --quiet --filter=blob:none --no-checkout --depth 1 --branch $S.Branch $S.Git $tmp
        if ($LASTEXITCODE -ne 0) { throw "git clone exited $LASTEXITCODE" }
        & git @config -C $tmp sparse-checkout set --no-cone @($patterns)
        if ($LASTEXITCODE -ne 0) { throw "git sparse-checkout exited $LASTEXITCODE" }
        & git @config -C $tmp checkout --quiet $S.Branch
        if ($LASTEXITCODE -ne 0) { throw "git checkout exited $LASTEXITCODE" }

        # "H <path>" is an entry the sparse checkout kept, "S <path>" one it
        # left out; -s gives each entry's blob id, in the same order.
        $tags = @(& git @config -C $tmp ls-files -t)
        $stages = @(& git @config -C $tmp ls-files -s)
        if ($LASTEXITCODE -ne 0 -or $tags.Count -ne $stages.Count) { throw 'git ls-files did not list the index' }
        $expected = [ordered]@{}
        for ($i = 0; $i -lt $tags.Count; $i++) {
            if ($tags[$i].StartsWith('H ')) {
                $fields = $stages[$i] -split "`t", 2
                $expected[$fields[1]] = ($fields[0] -split ' ')[1]
            }
        }
        $paths = @($expected.Keys)
        if ($paths.Count -eq 0) { throw "nothing in $($S.Git) matches $($S.Keep -join ', ')" }

        # A blob id is the SHA-1 of "blob <length>", a NUL, and the bytes, so
        # hashing the bytes on disk gives the id only if checkout wrote the
        # committed bytes unchanged. Hashed here rather than by git hash-object,
        # because Windows PowerShell puts a byte order mark in front of
        # anything it pipes to a native program.
        $sha1 = [System.Security.Cryptography.SHA1]::Create()
        $buffer = New-Object byte[] (1MB)
        $wrong = @(foreach ($path in $paths) {
            $full = Join-Path $tmp $path
            if (-not (Test-Path -LiteralPath $full -PathType Leaf)) { $path; continue }
            $stream = [System.IO.File]::OpenRead($full)
            try {
                $sha1.Initialize()
                $header = [System.Text.Encoding]::ASCII.GetBytes("blob $($stream.Length)`0")
                [void]$sha1.TransformBlock($header, 0, $header.Length, $null, 0)
                while (($read = $stream.Read($buffer, 0, $buffer.Length)) -gt 0) {
                    [void]$sha1.TransformBlock($buffer, 0, $read, $null, 0)
                }
                [void]$sha1.TransformFinalBlock($buffer, 0, 0)
            }
            finally {
                $stream.Dispose()
            }
            $id = -join ($sha1.Hash | ForEach-Object { $_.ToString('x2') })
            if ($id -ne $expected[$path]) { $path }
        })
        if ($wrong.Count -gt 0) {
            throw "$($wrong.Count) files are missing or not the bytes committed, the first $($wrong[0])"
        }

        $commit = (& git -C $tmp rev-parse HEAD).Trim()
        $when = (& git -C $tmp log -1 --format=%cI HEAD).Trim()
        Remove-Item -LiteralPath (Join-Path $tmp '.git') -Recurse -Force
        [System.IO.File]::WriteAllLines((Join-Path $tmp '_revision.txt'),
            [string[]]@("$($S.Git) $($S.Branch) $commit $when", "$($paths.Count) files, each checked against its blob id: those matching $($S.Keep -join ' '), and $($patterns.Count - @($S.Keep).Count) certificates and keys the passwords table names"),
            (New-Object System.Text.UTF8Encoding $false))

        if (Test-Path -LiteralPath $Dest) { Remove-Item -LiteralPath $Dest -Recurse -Force }
        New-Item -ItemType Directory -Force -Path (Split-Path -Parent $Dest) | Out-Null
        Move-Item -LiteralPath $tmp -Destination $Dest
        Write-Host "$($S.Name) -- $($paths.Count) files at $commit, every one the bytes committed"
    }
    finally {
        [Console]::OutputEncoding = $savedConsole
        if (Test-Path -LiteralPath $tmp) {
            Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}

function Get-ApiSuite {
    param([object]$S, [string]$Dest)

    $headers = @{ 'User-Agent' = 'pdfbox-go-fetch-corpus' }
    $entries = Invoke-RestMethod -Uri $S.Api -Headers $headers -TimeoutSec 120

    # Staged and then moved into place, for the two reasons the archive path is:
    # a suite that is half downloaded when the network gives out must not look
    # complete to the next run, whose only check is that the directory exists;
    # and -Force has to answer with what is upstream now, not with that plus
    # whatever was deleted or renamed there since.
    $tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("corpus-" + [guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Force -Path $tmp | Out-Null
    try {
        foreach ($e in $entries) {
            if ($e.type -ne 'file') { continue }
            Invoke-WebRequest -Uri $e.download_url -OutFile (Join-Path $tmp $e.name) `
                -TimeoutSec 300 -UseBasicParsing
        }

        if (Test-Path -LiteralPath $Dest) { Remove-Item -LiteralPath $Dest -Recurse -Force }
        New-Item -ItemType Directory -Force -Path (Split-Path -Parent $Dest) | Out-Null
        Move-Item -LiteralPath $tmp -Destination $Dest
    }
    finally {
        Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
    }
}

$missingLinks = 0
foreach ($name in $Suite) {
    $s = $all | Where-Object { $_.Name -eq $name } | Select-Object -First 1
    $dest = Join-Path $corpusRoot $name
    $hasLinks = $s.PSObject.Properties.Name -contains 'LinkManifest'
    $hasPasswords = @($s.PSObject.Properties.Name | Where-Object { $_ -in 'PasswordManifest', 'Passwords', 'PasswordTable' }).Count -gt 0

    if ((Test-Path -LiteralPath $dest) -and -not $Force) {
        $have = (Get-ChildItem -LiteralPath $dest -Recurse -File -Filter *.pdf -ErrorAction SilentlyContinue).Count
        "$name -- already present, $have PDFs (pass -Force to refetch)"
        if ($hasLinks) {
            # a present suite may still be missing linked files; this fetches
            # only those
            $missingLinks += Resolve-Links -S $s -Dest $dest
        }
        if ($hasPasswords) { Write-Passwords -S $s -Dest $dest }
        continue
    }

    "$name -- fetching ~$([math]::Round($s.Bytes / 1MB)) MB ..."
    if ($s.PSObject.Properties.Name -contains 'Api') {
        Get-ApiSuite -S $s -Dest $dest
    }
    elseif ($s.PSObject.Properties.Name -contains 'Git') {
        Get-GitSuite -S $s -Dest $dest
    }
    else {
        Get-ArchiveSuite -S $s -Dest $dest
    }
    if ($hasLinks) {
        $missingLinks += Resolve-Links -S $s -Dest $dest
    }
    if ($hasPasswords) { Write-Passwords -S $s -Dest $dest }

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

if ($missingLinks -gt 0) {
    ''
    "$missingLinks linked file(s) could not be fetched with the right checksum; each is named in its suite's _links.tsv"
    exit 1
}
