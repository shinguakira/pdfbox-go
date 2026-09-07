package util

// Exposes PDFBox version.
//
// Port of org.apache.pdfbox.util.Version.
//
// Java reads `/org/apache/pdfbox/resources/version.properties` off the class
// path and answers its `pdfbox.version` key, or null where the resource cannot
// be read. That file holds `pdfbox.version=${project.version}`, a placeholder
// Maven substitutes when it copies the resource into the jar, so the value the
// running Java answers is the version in `pom.xml` and not the literal in the
// file. There is no Maven here and nothing substitutes anything, so the port
// carries the number itself and this comment says where it comes from.
//
// **When the Java side is updated, this constant is what has to move with it.**

// pdfBoxVersion is `<version>` of `pom.xml`, which Maven writes into
// version.properties as `pdfbox.version`.
const pdfBoxVersion = "4.0.0-SNAPSHOT"

// Version returns the version of PDFBox.
//
// Port of getVersion(), which answers null where the properties file is missing
// or unreadable. That cannot happen here -- the value is a constant, not a
// resource -- so the second return, which is Java's null, is always true. It is
// kept because the one caller, the `version` command, prints "unknown" for null
// and dropping it would drop that branch.
func Version() (string, bool) {
	return pdfBoxVersion, true
}
