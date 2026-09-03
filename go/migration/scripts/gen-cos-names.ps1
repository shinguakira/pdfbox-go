<#
.SYNOPSIS
    Generates go/pdfbox/cos/names.go from the constants in COSName.java.

.DESCRIPTION
    COSName.java declares ~590 predefined name constants, all of the form

        public static final COSName SCREAMING_SNAKE = getPDFName("PdfName");

    Transcribing those by hand would be error-prone in a way no reviewer could
    catch, so they are generated. The Go identifier comes from the PDF name
    string, not from the Java constant name, because the PDF string is what the
    value actually is; the Java identifier is only a label. Where that would
    produce an invalid or colliding Go identifier the mapping below fixes it.

    Regenerate after changing the mapping. The Java tree is frozen, so the
    input does not change on its own.

.EXAMPLE
    pwsh go/migration/scripts/gen-cos-names.ps1
#>
[CmdletBinding()]
param(
    [string]$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..\..')).Path
)

$ErrorActionPreference = 'Stop'

$javaFile = Join-Path $RepoRoot 'pdfbox\src\main\java\org\apache\pdfbox\cos\COSName.java'
$outFile  = Join-Path $RepoRoot 'go\pdfbox\cos\names.go'

# Identifiers keyed by the Java constant, for the cases the derivation rule
# below cannot resolve on its own.
#
# Most are case-sensitive PDF name pairs -- "CA" and "ca" are different names
# (stroking and non-stroking alpha), and upper-casing the first letter collapses
# them. For those the Go identifier mirrors the PDF name's own casing.
#
# Note the ordinal comparer: PowerShell hashtables are case-insensitive by
# default, which would collapse the OFF/Off keys below -- the very collision
# this table exists to resolve.
$javaOverrides = [System.Collections.Hashtable]::new([System.StringComparer]::Ordinal)
$javaOverrides['EMPTY']                     = 'Empty' # PDF name is "", which derives to nothing
$javaOverrides['CA']                        = 'CA'    # "CA", stroking alpha
$javaOverrides['CA_NS']                     = 'Ca'    # "ca", non-stroking alpha
$javaOverrides['OP']                        = 'OP'    # "OP"
$javaOverrides['OP_NS']                     = 'Op'    # "op"
$javaOverrides['OFF']                       = 'OFF'   # "OFF"
$javaOverrides['Off']                       = 'Off'   # "Off"
$javaOverrides['FL']                        = 'FL'    # "FL"
$javaOverrides['FLATE_DECODE_ABBREVIATION'] = 'Fl'    # "Fl"

# Go identifiers that would collide with a type or value already declared in
# package cos.
$identOverrides = [System.Collections.Hashtable]::new([System.StringComparer]::Ordinal)
$identOverrides['Name']   = 'NameKey'    # Name is the type
$identOverrides['True']   = 'TrueName'   # True is the boolean value
$identOverrides['False']  = 'FalseName'  # False is the boolean value
$identOverrides['Null']   = 'NullName'   # Null is the type
$identOverrides['String'] = 'StringName' # String is a method name on every type here
$identOverrides['Base']       = 'BaseName'       # Base is the interface
$identOverrides['Document']   = 'DocumentName'   # Document is the type
$identOverrides['Array']      = 'ArrayName'      # Array is the type
$identOverrides['Dictionary'] = 'DictionaryName' # Dictionary is the type
$identOverrides['Object']     = 'ObjectName'     # Object is the type
$identOverrides['Stream']     = 'StreamName'     # Stream is the type
$identOverrides['Integer']    = 'IntegerName'    # Integer is the type
$identOverrides['Float']      = 'FloatName'      # Float is the type
$identOverrides['Boolean']    = 'BooleanName'    # Boolean is the type
$identOverrides['Number']     = 'NumberName'     # Number is the interface

$lines = Get-Content -LiteralPath $javaFile
$entries = [System.Collections.Generic.List[object]]::new()
$seen = [System.Collections.Hashtable]::new([System.StringComparer]::Ordinal)

foreach ($line in $lines) {
    if ($line -notmatch 'public static final COSName\s+([A-Z0-9_]+)\s*=\s*getPDFName\("([^"]*)"\)') {
        continue
    }
    $javaConst = $Matches[1]
    $pdfName   = $Matches[2]

    if ($javaOverrides.ContainsKey($javaConst)) {
        $ident = $javaOverrides[$javaConst]
    }
    else {
        # Build a Go identifier from the PDF name: strip anything not
        # alphanumeric, upper-case the first letter of each remaining run.
        $ident = ($pdfName -split '[^A-Za-z0-9]' | Where-Object { $_ -ne '' } | ForEach-Object {
            $_.Substring(0, 1).ToUpperInvariant() + $_.Substring(1)
        }) -join ''

        if ($ident -match '^[0-9]') {
            # names like "3DD" cannot start a Go identifier
            $ident = 'N' + $ident
        }
        if ($identOverrides.ContainsKey($ident)) { $ident = $identOverrides[$ident] }
    }

    if ($ident -eq '') {
        throw "empty Go identifier for java $javaConst (pdf '$pdfName') - add a javaOverrides entry"
    }
    if ($seen.ContainsKey($ident)) {
        # Never skip: a dropped name is a silently missing constant, and the
        # colliding pair are usually distinct case-sensitive PDF names.
        throw "duplicate Go identifier '$ident' (java $javaConst, pdf '$pdfName') collides with java $($seen[$ident]) - add a javaOverrides entry"
    }
    $seen[$ident] = $javaConst

    $entries.Add([PSCustomObject]@{ Ident = $ident; PdfName = $pdfName; JavaConst = $javaConst })
}

$sorted = $entries | Sort-Object Ident

$body = [System.Collections.Generic.List[string]]::new()
$body.Add('// Code generated by migration/scripts/gen-cos-names.ps1. DO NOT EDIT.')
$body.Add('')
$body.Add('package cos')
$body.Add('')
$body.Add('// The predefined names from org.apache.pdfbox.cos.COSName.')
$body.Add('//')
$body.Add('// The Go identifier is derived from the PDF name string rather than from the')
$body.Add('// Java constant, since the string is what the value is. A few are renamed to')
$body.Add('// avoid colliding with a type or value already in this package; those carry a')
$body.Add('// comment saying so.')
$body.Add('var (')

foreach ($e in $sorted) {
    $natural = ($e.PdfName -split '[^A-Za-z0-9]' | Where-Object { $_ -ne '' } | ForEach-Object {
        $_.Substring(0, 1).ToUpperInvariant() + $_.Substring(1) }) -join ''
    $comment = if ($e.Ident -ne $natural) { " // java COSName.$($e.JavaConst)" } else { '' }
    $body.Add("`t$($e.Ident) = GetPDFName(`"$($e.PdfName)`")$comment")
}

$body.Add(')')

$utf8NoBom = New-Object System.Text.UTF8Encoding $false
[System.IO.File]::WriteAllLines($outFile, [string[]]$body, $utf8NoBom)

Write-Host "wrote $($sorted.Count) predefined names to $outFile"

# The var block above is not aligned the way gofmt wants, so format in place
# rather than leaving the tree dirty for the next `gofmt -l` check.
& gofmt -w $outFile
if ($LASTEXITCODE -ne 0) { throw "gofmt failed on $outFile" }
Write-Host "gofmt applied"
