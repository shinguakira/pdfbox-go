# Fills pdfbox/target/test-output/flatten/in, which PDAcroFormFlattenTest reads.
#
# That class is the one Java test suite that does not take its input from the
# poms. It carries its own list of JIRA attachment URLs in the source and
# downloads them itself at run time, into a directory under target/ that the
# Maven build never touches. fetch-testdata.ps1 reads the poms, so it does not
# see these, and a checkout that has run it still cannot run the flatten tests.
#
# The list below is read out of the test source rather than copied by hand:
# ten entries in its String[] and two more written inline in
# flattenTestPDFBOX5254 and flattenTestPDFBOX5225. Re-derive it with
#
#   grep -E '^\s+"https://issues' PDAcroFormFlattenTest.java
#   grep -A1 'String sourceUrl = "https' PDAcroFormFlattenTest.java
#
# if the class gains a case. Entries the class has commented out are left out
# here too, and the count below is the check on that.
#
#   pwsh -File go/migration/scripts/fetch-flatten.ps1
#
# Nothing here is committed: target/ is gitignored, and these are other
# projects' documents under other projects' licences.

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent (Split-Path -Parent (Split-Path -Parent $PSScriptRoot))
$dest = Join-Path $root 'pdfbox\target\test-output\flatten\in'
New-Item -ItemType Directory -Force -Path $dest | Out-Null

$list = @(
  'https://issues.apache.org/jira/secure/attachment/12682897/FormI-9-English.pdf,FormI-9-English.pdf',
  'https://issues.apache.org/jira/secure/attachment/12689788/test.pdf,test-2586.pdf',
  'https://issues.apache.org/jira/secure/attachment/12792007/hidden_fields.pdf,hidden_fields.pdf',
  'https://issues.apache.org/jira/secure/attachment/12816014/Signed-Document-1.pdf,Signed-Document-1.pdf',
  'https://issues.apache.org/jira/secure/attachment/12816016/Signed-Document-2.pdf,Signed-Document-2.pdf',
  'https://issues.apache.org/jira/secure/attachment/12821307/Signed-Document-3.pdf,Signed-Document-3.pdf',
  'https://issues.apache.org/jira/secure/attachment/12821308/Signed-Document-4.pdf,Signed-Document-4.pdf',
  'https://issues.apache.org/jira/secure/attachment/12986337/stenotypeTest-3_rotate_no_flatten.pdf,PDFBOX-4693-filled.pdf',
  'https://issues.apache.org/jira/secure/attachment/12994791/flatten.pdf,PDFBOX-4788.pdf',
  'https://issues.apache.org/jira/secure/attachment/13011410/PDFBOX-4955.pdf,PDFBOX-4955.pdf',
  'https://issues.apache.org/jira/secure/attachment/13005793/f1040sb%20test.pdf,PDFBOX-4889-5254.pdf',
  'https://issues.apache.org/jira/secure/attachment/13027311/SourceFailure.pdf,PDFBOX-5225.pdf'
)

$got = 0
$had = 0
$failed = 0
foreach ($entry in $list) {
  $parts = $entry.Split(',')
  $url = $parts[0]
  $name = $parts[1]
  $out = Join-Path $dest $name
  if (Test-Path $out) {
    $had++
    continue
  }
  try {
    Invoke-WebRequest -Uri $url -OutFile $out -UseBasicParsing -TimeoutSec 60
    # A JIRA attachment that has gone away answers with an HTML page, and an
    # HTML page is not a PDF. Check rather than leave a file that fails later
    # with a parse error nobody traces back to here.
    $head = [System.IO.File]::ReadAllBytes($out)[0..4]
    if (-not ($head[0] -eq 0x25 -and $head[1] -eq 0x50 -and $head[2] -eq 0x44 -and $head[3] -eq 0x46)) {
      Remove-Item $out -Force
      throw 'the response does not start with %PDF'
    }
    $got++
    '{0,-32} {1,9:N0} bytes' -f $name, (Get-Item $out).Length
  } catch {
    $failed++
    '{0,-32} FAILED: {1}' -f $name, $_.Exception.Message
  }
}
''
'fetched {0}, already present {1}, failed {2}, of {3} declared' -f $got, $had, $failed, $list.Count
if ($failed -gt 0) { exit 1 }
