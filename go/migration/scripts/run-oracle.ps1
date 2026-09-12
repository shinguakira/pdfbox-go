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

    The Java tree is not touched. The only .java file this repository owns is
    migration/oracle/JavaCorpus.java, which sits outside the Maven module
    directories and compiles against them.

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

.PARAMETER Rebuild
    Recompile even when the classes are already there.

.EXAMPLE
    pwsh go/migration/scripts/run-oracle.ps1

.EXAMPLE
    pwsh go/migration/scripts/run-oracle.ps1 -List failures.txt -Out java.tsv
#>
[CmdletBinding()]
param(
    [string]$List,

    [string]$Out,

    [switch]$Crlf,

    [int]$TimeoutSeconds = 20,

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
        (Join-Path $RepoRoot 'pdfbox/src/main/java')
    ) -Recurse -Filter *.java -File | ForEach-Object { $_.FullName }

    # Not Set-Content: Windows PowerShell writes UTF-8 with a byte order mark,
    # and javac's argfile parser reads the mark as part of the first path and
    # exits 3.
    Write-NoBom -Path $sourceList -Lines $sources

    $count = $sources.Count
    Write-Host "compiling $count Java files"
    & javac -nowarn -encoding UTF-8 -d $classes -cp $classpath "@$sourceList"
    if ($LASTEXITCODE -ne 0) { throw "javac exited $LASTEXITCODE" }

    & javac -nowarn -encoding UTF-8 -d $classes -cp $classpath `
        (Join-Path $RepoRoot 'go/migration/oracle/JavaCorpus.java')
    if ($LASTEXITCODE -ne 0) { throw "javac exited $LASTEXITCODE on the driver" }
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

$total = (Get-Content -LiteralPath $List).Count
Write-Host "running PDFBox over $total files"

# -Xss8m because a recursive page tree reaches StackOverflowError at the default
# and the point is to see what PDFBox does, not what its stack size does.
Push-Location $RepoRoot
try {
    $lfArg = if ($Crlf) { 'crlf' } else { 'lf' }
    $rows = & java -Xss8m -cp $classpath JavaCorpus $List $TimeoutSeconds $lfArg
    if ($LASTEXITCODE -ne 0) { throw "java exited $LASTEXITCODE" }
    Write-NoBom -Path $Out -Lines $rows
}
finally {
    Pop-Location
}

''
"wrote $Out"
"compare with:  go run ./cmd/corpus -oracle `"$Out`" ./testdata/corpus ../pdfbox/target/pdfs ../examples/target/pdfs"
