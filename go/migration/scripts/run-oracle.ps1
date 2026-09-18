<#
.SYNOPSIS
    Runs PDFBox over a list of PDFs and writes the table cmd/corpus writes, so
    the port can be compared against the Java it is a port of.

.DESCRIPTION
    migration/README.md says PDFBox is the oracle and the specification is only
    the tiebreaker. That has meant one file at a time, in a debugger. This runs
    it over a corpus.

    It needs a JDK and nothing else -- no Maven. javac compiles io, fontbox and
    pdfbox straight out of the tree, and the compile-scope jars the poms name
    (Bouncy Castle, log4j-api) are fetched once into the cache directory and
    reused. The resource directories go on the classpath rather than being
    copied, because that is all PDFBox needs to find its glyph lists, AFMs and
    predefined CMaps.

    The Java tree is not touched. The only .java files this repository owns are
    the two in migration/oracle: JavaCorpus.java, the driver this script runs,
    and JavaBench.java, the Java half of migration/BENCHMARK.md. Both sit
    outside the Maven module directories and compile against them, and both
    are compiled here.

    The output is a TSV with the same columns cmd/corpus emits. Feed the two to

        go run ./cmd/corpus -oracle java.tsv <dirs>

    which reports the disagreements.

.PARAMETER List
    A file of repository-relative PDF paths, one per line. Defaults to every
    PDF under the directories cmd/corpus is normally pointed at.

.PARAMETER Out
    Where to write the table. Defaults to java-corpus.tsv in the cache.

.PARAMETER Crlf
    Leave the line separator at System.lineSeparator() instead of forcing LF.
    Only useful for demonstrating what it costs: on Windows every document with
    text in it then differs from the port by one character per line.

.PARAMETER TimeoutSeconds
    Give up on one file after this. Default 20, matching cmd/corpus.

.PARAMETER Pages
    Write a digest per page instead of a row per file. The table is
    file, page, text, chars, digest, and `corpus -comparepages` joins it with
    the one `corpus -pages` writes. Use it over the files a comparison has
    already named, not over a corpus: it extracts each page in a pass of its
    own.

.PARAMETER Passwords
    Tables of the ways encrypted files are opened, in UTF-8 without a byte order
    mark. Each line is one way of opening one file: a path ending, a tab, and the
    password; or a path ending, the password, a certificate and a private key,
    tab separated, the two files named relative to the table's directory; or a
    path ending, a passphrase and a PKCS#12 keystore, for a project that keeps
    that certificate and key in a store of its own, which is the shape PDFBox
    reads and is handed to it as it is. A file
    is opened, and gets a row, once for every line naming it. Pass the same
    tables to cmd/corpus as -passwords, in the same order, so both sides open the
    same files the same ways. fetch-corpus.ps1 writes one for a suite that
    publishes its passwords, as _passwords.tsv beside the files.

.PARAMETER Rebuild
    Recompile even when the classes are already there.

.EXAMPLE
    pwsh go/migration/scripts/run-oracle.ps1

.EXAMPLE
    pwsh go/migration/scripts/run-oracle.ps1 -List failures.txt -Out java.tsv

.EXAMPLE
    pwsh go/migration/scripts/run-oracle.ps1 -List pdfjs.txt -Out java-pdfjs.tsv -Passwords go/testdata/corpus/pdfjs/_passwords.tsv
#>
[CmdletBinding()]
param(
    [string]$List,

    [string]$Out,

    [switch]$Crlf,

    [int]$TimeoutSeconds = 20,

    [string[]]$Passwords,

    # Write a digest per page rather than a row per file, for narrowing a
    # disagreement the document digest found to a page. go/cmd/corpus -pages
    # writes the same table for the port, and -comparepages joins the two.
    [switch]$Pages,

    # Write a digest per facet of each document beyond its text -- positions,
    # document information, XMP, outline, page labels, page boxes, structure
    # tree, annotations, form fields, images -- instead of the row table.
    # go/cmd/corpus -facets writes the same table, and -comparefacets joins them.
    [switch]$Facets,

    # Write every line the facet digests are made of, for narrowing a facet two
    # tables disagree on. Run it over the files a comparison named.
    [switch]$FacetLines,

    # Write what PDFBox writes, read back: save, incremental save, encryption on
    # saving, split, merge, overlay and an external signature, each summarised
    # as the page count and text digest of what it wrote (JavaWrites).
    # go/cmd/corpus -writes writes the same table, and -comparefacets joins them.
    [switch]$Writes,

    # Write every line the write digests are made of.
    [switch]$WriteLines,

    # Render every page at 72 dpi and write its size, a digest of its pixels and
    # a 16 by 16 grid of mean brightness (JavaRender). go/cmd/corpus -render
    # writes the same table, and -comparerender joins them.
    [switch]$Render,

    [switch]$Rebuild,

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

foreach ($tool in 'javac', 'java') {
    if (-not (Get-Command $tool -ErrorAction SilentlyContinue)) {
        throw "$tool is not on PATH; this needs a JDK (11 or later, as the poms ask)"
    }
}

# Everything this builds is a build product and none of it is committed.
$cache = Join-Path $RepoRoot 'go/testdata/oracle'
$lib = Join-Path $cache 'lib'
$classes = Join-Path $cache 'classes'
New-Item -ItemType Directory -Force -Path $lib, $classes | Out-Null

if (-not $Out) { $Out = Join-Path $cache 'java-corpus.tsv' }

# Windows PowerShell has no BOM-less UTF-8 encoding to pass to Set-Content, and
# both javac and the driver read these files as plain paths.
function Write-NoBom {
    param([string]$Path, [string[]]$Lines)
    [System.IO.File]::WriteAllLines($Path, $Lines, (New-Object System.Text.UTF8Encoding $false))
}

# --------------------------------------------------------------------- the jars

# The versions the poms name. bcpkix and bcutil have no 1.85.2 -- Bouncy Castle
# patches bcprov on its own -- so the BOM's 1.85 is what is on Maven Central.
$jars = @(
    @{ Name = 'bcprov-jdk18on-1.85.2.jar'; Path = 'org/bouncycastle/bcprov-jdk18on/1.85.2' }
    @{ Name = 'bcpkix-jdk18on-1.85.jar'; Path = 'org/bouncycastle/bcpkix-jdk18on/1.85' }
    @{ Name = 'bcutil-jdk18on-1.85.jar'; Path = 'org/bouncycastle/bcutil-jdk18on/1.85' }
    @{ Name = 'log4j-api-2.26.1.jar'; Path = 'org/apache/logging/log4j/log4j-api/2.26.1' }
)

foreach ($jar in $jars) {
    $target = Join-Path $lib $jar.Name
    if (Test-Path -LiteralPath $target) { continue }
    $url = "https://repo1.maven.org/maven2/$($jar.Path)/$($jar.Name)"
    Write-Host "fetching $($jar.Name)"
    Invoke-WebRequest -Uri $url -OutFile $target -TimeoutSec 300 -UseBasicParsing
}

# The resource directories go on the classpath because PDFBox loads its glyph
# lists, AFMs and predefined CMaps off it, and javac does not copy them.
$classpath = (@($classes) +
    @((Join-Path $RepoRoot 'pdfbox/src/main/resources'), (Join-Path $RepoRoot 'fontbox/src/main/resources')) +
    @(Get-ChildItem -LiteralPath $lib -Filter *.jar | ForEach-Object { $_.FullName })) -join
    [System.IO.Path]::PathSeparator

# ------------------------------------------------------------------ the compile

$driver = Join-Path $classes 'JavaCorpus.class'
if ($Rebuild -or -not (Test-Path -LiteralPath $driver)) {
    $sourceList = Join-Path $cache 'sources.txt'
    $sources = Get-ChildItem -Path @(
        (Join-Path $RepoRoot 'io/src/main/java'),
        (Join-Path $RepoRoot 'fontbox/src/main/java'),
        (Join-Path $RepoRoot 'pdfbox/src/main/java'),
        (Join-Path $RepoRoot 'xmpbox/src/main/java')
    ) -Recurse -Filter *.java -File | ForEach-Object { $_.FullName }

    # Not Set-Content: Windows PowerShell writes UTF-8 with a byte order mark,
    # and javac's argfile parser reads the mark as part of the first path and
    # exits 3.
    Write-NoBom -Path $sourceList -Lines $sources

    $count = $sources.Count
    Write-Host "compiling $count Java files"
    & javac -nowarn -encoding UTF-8 -d $classes -cp $classpath "@$sourceList"
    if ($LASTEXITCODE -ne 0) { throw "javac exited $LASTEXITCODE" }
}

# The two drivers are small and they change, so each is compiled whenever its
# class is missing or older than its source -- not only when the tree is. A
# driver compiled once and never again would go on running the old code after
# an edit, with nothing to say so.
# JavaCorpus and JavaFacets name each other, so each is compiled with the
# directory as its source path and javac finds the other.
$oracleSources = Join-Path $RepoRoot 'go/migration/oracle'
foreach ($name in 'JavaCorpus', 'JavaFacets', 'JavaWrites', 'JavaRender', 'JavaBench') {
    $source = Join-Path $oracleSources "$name.java"
    $class = Join-Path $classes "$name.class"
    $stale = $Rebuild -or -not (Test-Path -LiteralPath $class) -or
        ((Get-Item -LiteralPath $source).LastWriteTimeUtc -gt (Get-Item -LiteralPath $class).LastWriteTimeUtc)
    if ($stale) {
        Write-Host "compiling $name"
        & javac -nowarn -encoding UTF-8 -d $classes -cp $classpath -sourcepath $oracleSources $source
        if ($LASTEXITCODE -ne 0) { throw "javac exited $LASTEXITCODE on $name" }
    }
}

# --------------------------------------------------------------------- the list

if (-not $List) {
    $List = Join-Path $cache 'list.txt'
    $roots = @(
        'go/testdata/corpus'
        'pdfbox/target/pdfs'
        'examples/target/pdfs'
    ) | Where-Object { Test-Path -LiteralPath (Join-Path $RepoRoot $_) }

    if ($roots.Count -eq 0) {
        throw 'nothing to score -- run fetch-testdata.ps1 and fetch-corpus.ps1 first'
    }

    $paths = foreach ($root in $roots) {
        Get-ChildItem -LiteralPath (Join-Path $RepoRoot $root) -Recurse -File |
            Where-Object { $_.Extension -ieq '.pdf' } |
            ForEach-Object { $_.FullName.Substring($RepoRoot.Length + 1).Replace('\', '/') }
    }
    Write-NoBom -Path $List -Lines $paths
}

# Resolved here, against the directory the script was called from, because the
# run below moves to the repository root.
if ($Passwords) { $Passwords = @($Passwords | ForEach-Object { (Resolve-Path -LiteralPath $_).Path }) }

$total = (Get-Content -LiteralPath $List).Count
Write-Host "running PDFBox over $total files"

# -Xss8m because a recursive page tree reaches StackOverflowError at the default
# and the point is to see what PDFBox does, not what its stack size does.
Push-Location $RepoRoot
try {
    $lfArg = if ($Crlf) { 'crlf' } else { 'lf' }
    $mode = if ($Pages) { "pages" } elseif ($FacetLines) { "facetlines" } elseif ($Facets) { "facets" } elseif ($WriteLines) { "writelines" } elseif ($Writes) { "writes" } elseif ($Render) { "render" } else { "rows" }
    $javaArgs = @($List, $TimeoutSeconds, $lfArg, $mode)
    if ($Passwords) { $javaArgs += $Passwords }
    # JavaCorpus writes its table in UTF-8, and PowerShell reads what a native
    # program writes in the console's code page unless told otherwise; a row
    # named by a password in Arabic came back as question marks.
    $savedEncoding = [Console]::OutputEncoding
    [Console]::OutputEncoding = New-Object System.Text.UTF8Encoding $false
    try {
        $rows = & java -Xss8m -cp $classpath JavaCorpus @javaArgs
    }
    finally {
        [Console]::OutputEncoding = $savedEncoding
    }
    if ($LASTEXITCODE -ne 0) { throw "java exited $LASTEXITCODE" }
    Write-NoBom -Path $Out -Lines $rows
}
finally {
    Pop-Location
}

''
"wrote $Out"
"compare with:  go run ./cmd/corpus -oracle `"$Out`" ./testdata/corpus ../pdfbox/target/pdfs ../examples/target/pdfs"
